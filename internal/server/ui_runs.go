package server

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/muchobien/lyra/internal/store"
)

const runsPerPage = 50

type logGroup struct {
	Name   string
	Result string
	Step   *store.TaskStep
	Lines  []string
}

type runPage struct {
	Run      store.Run
	Job      store.Job
	Groups   []logGroup
	Lines    []string
	Active   bool
	CanWrite bool
	LogsURL  string
}

func uiRunNumber(r *http.Request) (int64, error) {
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		return 0, notFound("Run not found.")
	}
	return number, nil
}

func (s *Server) uiRuns(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	runs, err := s.store.Runs(ctx, repo.Name, runsPerPage)
	if err != nil {
		return err
	}
	ref := s.git.DefaultBranch(ctx, s.repoDir(repo.Name))
	s.render(w, r, http.StatusOK, "runs", s.repoPage(r, &repo, "Actions · "+repo.Name, ref, runs))
	return nil
}

func selectedJob(r *http.Request, jobs []store.Job) store.Job {
	selected, err := strconv.ParseInt(r.URL.Query().Get("job"), 10, 64)
	index := slices.IndexFunc(jobs, func(job store.Job) bool { return job.ID == selected })
	if err != nil || index < 0 {
		return jobs[0]
	}
	return jobs[index]
}

func (s *Server) showJobLog(r *http.Request, view *runPage) error {
	if view.Job.TaskID == 0 {
		return nil
	}
	task, _, err := s.store.Task(r.Context(), view.Job.TaskID)
	if err != nil {
		return err
	}
	lines, err := s.readLog(r.Context(), task, 0)
	if err != nil {
		return err
	}
	for _, line := range lines {
		view.Lines = append(view.Lines, line.Content)
	}
	view.Active = view.Active || (task.Status == statusRunning && task.LogObject == "")
	if !view.Active {
		view.Groups = logGroups(view.Job.Steps, view.Lines)
	}
	return nil
}

func (s *Server) uiRun(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	number, err := uiRunNumber(r)
	if err != nil {
		return err
	}
	run, err := s.store.Run(r.Context(), repo.Name, number)
	if err != nil {
		return missing(err, fmt.Sprintf("Run #%d not found.", number))
	}
	run = withStepNames(run)
	view := runPage{Run: run, CanWrite: identityFrom(r).can(scopeWrite, repo.Name)}
	if len(run.Jobs) > 0 {
		view.Job = selectedJob(r, run.Jobs)
		view.Active = slices.Contains(activeStatuses, view.Job.Status)
		view.LogsURL = fmt.Sprintf("/api/repos/%s/runs/%d/jobs/%d/logs", repo.Name, run.Number, view.Job.ID)
		if err := s.showJobLog(r, &view); err != nil {
			return err
		}
	}
	title := run.Name + " #" + fmt.Sprint(run.Number)
	s.render(w, r, http.StatusOK, "run", s.repoPage(r, &repo, title, run.SHA, view))
	return nil
}

func window(lines []string, from, to int64) []string {
	length := int64(len(lines))
	from, to = max(0, min(from, length)), max(0, min(to, length))
	if to <= from {
		return nil
	}
	return lines[from:to]
}

func logGroups(steps []store.TaskStep, lines []string) []logGroup {
	groups := []logGroup{}
	position := int64(0)
	for index := range steps {
		step := &steps[index]
		if step.LogLength == 0 && step.Result == "" {
			groups = append(groups, logGroup{Name: step.Name, Step: step})
			continue
		}
		if step.LogIndex > position {
			groups = append(groups, logGroup{Name: "Set up", Lines: window(lines, position, step.LogIndex)})
		}
		end := step.LogIndex + step.LogLength
		groups = append(groups, logGroup{
			Name:   step.Name,
			Result: step.Result,
			Step:   step,
			Lines:  window(lines, step.LogIndex, end),
		})
		position = end
	}
	if rest := window(lines, position, int64(len(lines))); len(rest) > 0 {
		groups = append(groups, logGroup{Name: "Complete job", Lines: rest})
	}
	return groups
}

func (s *Server) uiRunAction(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	id := identityFrom(r)
	if !id.can(scopeWrite, repo.Name) {
		return forbidden("This token can't change runs in " + repo.Name + ".")
	}
	number, err := uiRunNumber(r)
	if err != nil {
		return err
	}
	switch r.PathValue("action") {
	case "cancel":
		_, err = s.store.CancelRun(ctx, id.actor, repo.Name, number)
	case "rerun":
		_, err = s.store.RerunRun(ctx, id.actor, repo.Name, number, r.FormValue("failed") == "true")
	default:
		return notFound("Unknown action.")
	}
	if err != nil {
		return missing(err, fmt.Sprintf("Run #%d not found.", number))
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/runs/%d", repo.Name, number), http.StatusSeeOther)
	return nil
}
