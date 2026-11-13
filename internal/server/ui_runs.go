package server

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/muchobien/lyra/internal/store"
)

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

var activeStatuses = []string{"waiting", "blocked", "running"}

func (s *Server) uiRuns(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	runs, err := s.store.Runs(repo.Name, 50)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	s.render(w, r, http.StatusOK, "runs", page{Title: "Actions · " + repo.Name, Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: runs})
}

func (s *Server) uiRun(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "Run not found.")
		return
	}
	run, err := s.store.Run(repo.Name, number)
	if err != nil {
		s.renderStoreError(w, r, err, fmt.Sprintf("Run #%d not found.", number))
		return
	}
	run = withStepNames(run)
	view := runPage{Run: run, CanWrite: identityFrom(r).can("write", repo.Name)}
	if len(run.Jobs) == 0 {
		s.render(w, r, http.StatusOK, "run", page{Title: run.Name + " #" + fmt.Sprint(run.Number), Repo: &repo, Ref: run.SHA, CloneURL: cloneURL(r, repo.Name), Data: view})
		return
	}
	view.Job = run.Jobs[0]
	if selected, err := strconv.ParseInt(r.URL.Query().Get("job"), 10, 64); err == nil {
		if index := slices.IndexFunc(run.Jobs, func(job store.Job) bool { return job.ID == selected }); index >= 0 {
			view.Job = run.Jobs[index]
		}
	}
	view.Active = slices.Contains(activeStatuses, view.Job.Status)
	view.LogsURL = fmt.Sprintf("/api/repos/%s/runs/%d/jobs/%d/logs", repo.Name, run.Number, view.Job.ID)
	if view.Job.TaskID != 0 {
		task, _, err := s.store.Task(view.Job.TaskID)
		if err != nil {
			s.renderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		lines, err := readLog(r.Context(), s, task, 0)
		if err != nil {
			s.renderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		for _, line := range lines {
			view.Lines = append(view.Lines, line.Content)
		}
		view.Active = view.Active || task.Status == "running" && task.LogObject == ""
		if !view.Active {
			view.Groups = logGroups(view.Job.Steps, view.Lines)
		}
	}
	s.render(w, r, http.StatusOK, "run", page{Title: run.Name + " #" + fmt.Sprint(run.Number), Repo: &repo, Ref: run.SHA, CloneURL: cloneURL(r, repo.Name), Data: view})
}

func logGroups(steps []store.TaskStep, lines []string) []logGroup {
	slice := func(from, to int64) []string {
		from, to = max(0, min(from, int64(len(lines)))), max(0, min(to, int64(len(lines))))
		if to <= from {
			return nil
		}
		return lines[from:to]
	}
	groups := []logGroup{}
	position := int64(0)
	for index := range steps {
		step := &steps[index]
		if step.LogLength == 0 && step.Result == "" {
			groups = append(groups, logGroup{Name: step.Name, Step: step})
			continue
		}
		if step.LogIndex > position {
			groups = append(groups, logGroup{Name: "Set up", Lines: slice(position, step.LogIndex)})
		}
		groups = append(groups, logGroup{Name: step.Name, Result: step.Result, Step: step, Lines: slice(step.LogIndex, step.LogIndex+step.LogLength)})
		position = step.LogIndex + step.LogLength
	}
	if rest := slice(position, int64(len(lines))); len(rest) > 0 {
		groups = append(groups, logGroup{Name: "Complete job", Lines: rest})
	}
	return groups
}

func (s *Server) uiRunAction(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	id := identityFrom(r)
	if !id.can("write", repo.Name) {
		s.renderError(w, r, http.StatusForbidden, "This token can't change runs in "+repo.Name+".")
		return
	}
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "Run not found.")
		return
	}
	switch r.PathValue("action") {
	case "cancel":
		_, err = s.store.CancelRun(id.actor, repo.Name, number)
	case "rerun":
		_, err = s.store.RerunRun(id.actor, repo.Name, number, r.FormValue("failed") == "true")
	default:
		s.renderError(w, r, http.StatusNotFound, "Unknown action.")
		return
	}
	if err != nil {
		s.renderStoreError(w, r, err, fmt.Sprintf("Run #%d not found.", number))
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/runs/%d", repo.Name, number), http.StatusSeeOther)
}
