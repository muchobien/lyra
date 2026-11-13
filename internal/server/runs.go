package server

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"code.forgejo.org/forgejo/runner/v13/act/jobparser"

	"github.com/muchobien/lyra/internal/store"
)

func stepNames(payload string) []string {
	singles, err := jobparser.Parse([]byte(payload), false)
	if err != nil || len(singles) != 1 {
		return nil
	}
	_, job := singles[0].Job()
	names := []string{}
	for _, step := range job.Steps {
		name := step.Name
		switch {
		case name != "":
		case step.Uses != "":
			name = step.Uses
		default:
			line, _, _ := strings.Cut(strings.TrimSpace(step.Run), "\n")
			name = "Run " + line
		}
		names = append(names, name)
	}
	return names
}

func withStepNames(run store.Run) store.Run {
	for index, job := range run.Jobs {
		names := stepNames(job.Payload)
		for position, name := range names {
			if position < len(job.Steps) {
				job.Steps[position].Name = name
			} else {
				job.Steps = append(job.Steps, store.TaskStep{ID: int64(position), Name: name})
			}
		}
		run.Jobs[index] = job
	}
	return run
}

func runNumber(w http.ResponseWriter, r *http.Request) (int64, bool) {
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil || number <= 0 {
		writeError(w, http.StatusBadRequest, "invalid run number "+r.PathValue("number"))
		return 0, false
	}
	return number, true
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	limit, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 30
	}
	runs, err := s.store.Runs(repo.Name, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for index := range runs {
		runs[index] = withStepNames(runs[index])
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *Server) lookupRun(w http.ResponseWriter, r *http.Request, scope string) (store.Run, bool) {
	repo, ok := s.lookupRepo(w, r, scope)
	if !ok {
		return store.Run{}, false
	}
	number, ok := runNumber(w, r)
	if !ok {
		return store.Run{}, false
	}
	run, err := s.store.Run(repo.Name, number)
	if err != nil {
		writeStoreError(w, err, fmt.Sprintf("run #%d not found", number))
		return run, false
	}
	return withStepNames(run), true
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	run, ok := s.lookupRun(w, r, "read")
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleRunLogs(w http.ResponseWriter, r *http.Request) {
	run, ok := s.lookupRun(w, r, "read")
	if !ok {
		return
	}
	jobID, err := strconv.ParseInt(r.PathValue("job"), 10, 64)
	index := slices.IndexFunc(run.Jobs, func(job store.Job) bool { return job.ID == jobID })
	if err != nil || index < 0 {
		writeError(w, http.StatusNotFound, "job "+r.PathValue("job")+" not found in run #"+fmt.Sprint(run.Number))
		return
	}
	job := run.Jobs[index]
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	lines := []store.LogLine{}
	done := !slices.Contains([]string{"waiting", "blocked", "running"}, job.Status)
	if job.TaskID != 0 {
		task, _, err := s.store.Task(job.TaskID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if lines, err = readLog(r.Context(), s, task, max(offset, 0)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		done = done && (task.LogObject != "" || task.Status != "running")
	}
	next := max(offset, 0) + int64(len(lines))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Lyra-Log-Next", fmt.Sprint(next))
	w.Header().Set("Lyra-Log-Done", strconv.FormatBool(done))
	for _, line := range lines {
		fmt.Fprintln(w, line.Content)
	}
}

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	number, ok := runNumber(w, r)
	if !ok {
		return
	}
	eventID, err := s.store.CancelRun(identityFrom(r).actor, repo.Name, number)
	var invalid store.InvalidError
	if errors.As(err, &invalid) {
		writeError(w, http.StatusConflict, invalid.Error())
		return
	}
	if err != nil {
		writeStoreError(w, err, fmt.Sprintf("run #%d not found", number))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cancelled": number, "event_id": eventID})
}

func (s *Server) handleRerunRun(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	number, ok := runNumber(w, r)
	if !ok {
		return
	}
	eventID, err := s.store.RerunRun(identityFrom(r).actor, repo.Name, number, r.URL.Query().Get("failed") == "true")
	var invalid store.InvalidError
	if errors.As(err, &invalid) {
		writeError(w, http.StatusConflict, invalid.Error())
		return
	}
	if err != nil {
		writeStoreError(w, err, fmt.Sprintf("run #%d not found", number))
		return
	}
	run, err := s.store.Run(repo.Name, number)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, struct {
		store.Run
		EventID int64 `json:"event_id"`
	}{withStepNames(run), eventID})
}
