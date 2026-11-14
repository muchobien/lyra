package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"code.forgejo.org/forgejo/runner/v13/act/jobparser"

	"github.com/muchobien/lyra/internal/store"
)

const (
	statusWaiting   = "waiting"
	statusBlocked   = "blocked"
	statusRunning   = "running"
	statusSuccess   = "success"
	statusFailure   = "failure"
	statusCancelled = "cancelled"
	statusSkipped   = "skipped"

	checksNone    = "none"
	checksPending = "pending"
	checksFailure = "failure"
)

var activeStatuses = []string{statusWaiting, statusBlocked, statusRunning}

type runCancelled struct {
	Cancelled int64 `json:"cancelled"`
	EventID   int64 `json:"event_id"`
}

type runWrite struct {
	store.Run
	EventID int64 `json:"event_id"`
}

func stepName(step *jobparser.Step) string {
	switch {
	case step.Name != "":
		return step.Name
	case step.Uses != "":
		return step.Uses
	}
	line, _, _ := strings.Cut(strings.TrimSpace(step.Run), "\n")
	return "Run " + line
}

func stepNames(payload string) []string {
	singles, err := jobparser.Parse([]byte(payload), false)
	if err != nil || len(singles) != 1 {
		return nil
	}
	_, job := singles[0].Job()
	names := []string{}
	for _, step := range job.Steps {
		names = append(names, stepName(step))
	}
	return names
}

func withStepNames(run store.Run) store.Run {
	for index, job := range run.Jobs {
		for position, name := range stepNames(job.Payload) {
			if position < len(job.Steps) {
				job.Steps[position].Name = name
				continue
			}
			job.Steps = append(job.Steps, store.TaskStep{ID: int64(position), Name: name})
		}
		run.Jobs[index] = job
	}
	return run
}

type RunPath struct {
	RepoPath
	Number int64 `path:"number" minimum:"1"`
}

type listRunsInput struct {
	RepoPath
	Limit int `query:"limit" doc:"1 to 1000; defaults to 30."`
}

type runLogsInput struct {
	RunPath
	Job    int64 `path:"job"`
	Offset int64 `query:"offset" doc:"The first line to return."`
}

type runLogsOutput struct {
	ContentType string `header:"Content-Type"`
	Next        int64  `header:"Lyra-Log-Next" doc:"The offset to request next."`
	Done        bool   `header:"Lyra-Log-Done" doc:"Whether the job finished and its log is complete."`
	Body        []byte
}

type rerunInput struct {
	RunPath
	Failed bool `query:"failed" doc:"Rerun only the jobs that didn't succeed, and the jobs that need them."`
}

func runMissing(err error, number int64) error {
	return missing(err, fmt.Sprintf("run #%d not found", number))
}

func (s *Server) handleListRuns(r *http.Request, input *listRunsInput) (*body[[]store.Run], error) {
	repo, err := s.lookupRepo(r, scopeRead, input.fullName())
	if err != nil {
		return nil, err
	}
	runs, err := s.store.Runs(r.Context(), repo.Name, int64(queryLimit(input.Limit, 30, 1000)))
	if err != nil {
		return nil, err
	}
	for index := range runs {
		runs[index] = withStepNames(runs[index])
	}
	return reply(runs)
}

func (s *Server) lookupRun(r *http.Request, scope string, path RunPath) (store.Run, error) {
	repo, err := s.lookupRepo(r, scope, path.fullName())
	if err != nil {
		return store.Run{}, err
	}
	run, err := s.store.Run(r.Context(), repo.Name, path.Number)
	if err != nil {
		return run, runMissing(err, path.Number)
	}
	return withStepNames(run), nil
}

func (s *Server) handleGetRun(r *http.Request, input *RunPath) (*body[store.Run], error) {
	run, err := s.lookupRun(r, scopeRead, *input)
	if err != nil {
		return nil, err
	}
	return reply(run)
}

func (s *Server) jobLog(r *http.Request, job store.Job, offset int64) ([]store.LogLine, bool, error) {
	done := !slices.Contains(activeStatuses, job.Status)
	if job.TaskID == 0 {
		return []store.LogLine{}, done, nil
	}
	task, _, err := s.store.Task(r.Context(), job.TaskID)
	if err != nil {
		return nil, false, err
	}
	lines, err := s.readLog(r.Context(), task, offset)
	return lines, done && (task.LogObject != "" || task.Status != statusRunning), err
}

func (s *Server) handleRunLogs(r *http.Request, input *runLogsInput) (*runLogsOutput, error) {
	run, err := s.lookupRun(r, scopeRead, input.RunPath)
	if err != nil {
		return nil, err
	}
	index := slices.IndexFunc(run.Jobs, func(job store.Job) bool { return job.ID == input.Job })
	if index < 0 {
		return nil, notFound(fmt.Sprintf("job %d not found in run #%d", input.Job, run.Number))
	}
	offset := max(input.Offset, 0)
	lines, done, err := s.jobLog(r, run.Jobs[index], offset)
	if err != nil {
		return nil, err
	}
	var text strings.Builder
	for _, line := range lines {
		text.WriteString(line.Content + "\n")
	}
	return &runLogsOutput{
		ContentType: "text/plain; charset=utf-8",
		Next:        offset + int64(len(lines)),
		Done:        done,
		Body:        []byte(text.String()),
	}, nil
}

func (s *Server) handleCancelRun(r *http.Request, input *RunPath) (*body[runCancelled], error) {
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return nil, err
	}
	eventID, err := s.store.CancelRun(r.Context(), identityFrom(r).actor, repo.Name, input.Number)
	if err != nil {
		return nil, runMissing(invalidAsConflict(err), input.Number)
	}
	return reply(runCancelled{Cancelled: input.Number, EventID: eventID})
}

func (s *Server) handleRerunRun(r *http.Request, input *rerunInput) (*body[runWrite], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return nil, err
	}
	eventID, err := s.store.RerunRun(ctx, identityFrom(r).actor, repo.Name, input.Number, input.Failed)
	if err != nil {
		return nil, runMissing(invalidAsConflict(err), input.Number)
	}
	run, err := s.store.Run(ctx, repo.Name, input.Number)
	if err != nil {
		return nil, err
	}
	return reply(runWrite{Run: withStepNames(run), EventID: eventID})
}
