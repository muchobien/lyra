package server

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
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

func runNumber(r *http.Request) (int64, error) {
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil || number <= 0 {
		return 0, badRequest("invalid run number " + r.PathValue("number"))
	}
	return number, nil
}

func runMissing(err error, number int64) error {
	return missing(err, fmt.Sprintf("run #%d not found", number))
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	runs, err := s.store.Runs(r.Context(), repo.Name, int64(queryLimit(r, 30, 1000)))
	if err != nil {
		return err
	}
	for index := range runs {
		runs[index] = withStepNames(runs[index])
	}
	return writeJSON(w, http.StatusOK, runs)
}

func (s *Server) lookupRun(r *http.Request, scope string) (store.Run, error) {
	repo, err := s.lookupRepo(r, scope)
	if err != nil {
		return store.Run{}, err
	}
	number, err := runNumber(r)
	if err != nil {
		return store.Run{}, err
	}
	run, err := s.store.Run(r.Context(), repo.Name, number)
	if err != nil {
		return run, runMissing(err, number)
	}
	return withStepNames(run), nil
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) error {
	run, err := s.lookupRun(r, scopeRead)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, run)
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

func (s *Server) handleRunLogs(w http.ResponseWriter, r *http.Request) error {
	run, err := s.lookupRun(r, scopeRead)
	if err != nil {
		return err
	}
	jobID, err := strconv.ParseInt(r.PathValue("job"), 10, 64)
	index := slices.IndexFunc(run.Jobs, func(job store.Job) bool { return job.ID == jobID })
	if err != nil || index < 0 {
		return notFound("job " + r.PathValue("job") + " not found in run #" + fmt.Sprint(run.Number))
	}
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	offset = max(offset, 0)
	lines, done, err := s.jobLog(r, run.Jobs[index], offset)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Lyra-Log-Next", fmt.Sprint(offset+int64(len(lines))))
	w.Header().Set("Lyra-Log-Done", strconv.FormatBool(done))
	for _, line := range lines {
		fmt.Fprintln(w, line.Content)
	}
	return nil
}

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return err
	}
	number, err := runNumber(r)
	if err != nil {
		return err
	}
	eventID, err := s.store.CancelRun(r.Context(), identityFrom(r).actor, repo.Name, number)
	if err != nil {
		return runMissing(invalidAsConflict(err), number)
	}
	return writeJSON(w, http.StatusOK, runCancelled{Cancelled: number, EventID: eventID})
}

func (s *Server) handleRerunRun(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return err
	}
	number, err := runNumber(r)
	if err != nil {
		return err
	}
	failedOnly := r.URL.Query().Get("failed") == "true"
	eventID, err := s.store.RerunRun(ctx, identityFrom(r).actor, repo.Name, number, failedOnly)
	if err != nil {
		return runMissing(invalidAsConflict(err), number)
	}
	run, err := s.store.Run(ctx, repo.Name, number)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, runWrite{Run: withStepNames(run), EventID: eventID})
}
