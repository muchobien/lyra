package server

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"code.forgejo.org/forgejo/runner/v13/act/jobparser"
	"code.forgejo.org/forgejo/runner/v13/act/model"
	"github.com/robfig/cron/v3"

	"github.com/muchobien/lyra/internal/store"
)

func (s *Server) handleDispatch(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	var body struct {
		Workflow string            `json:"workflow"`
		Ref      string            `json:"ref"`
		Inputs   map[string]string `json:"inputs"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	dir := s.repoDir(repo.Name)
	if body.Ref == "" {
		body.Ref = s.git.DefaultBranch(dir)
	}
	ref, sha := "", ""
	for _, candidate := range []string{"refs/heads/" + body.Ref, "refs/tags/" + body.Ref, body.Ref} {
		if resolved, err := s.git.ResolveCommit(dir, candidate); err == nil && strings.HasPrefix(candidate, "refs/") {
			ref, sha = candidate, resolved
			break
		}
	}
	if sha == "" {
		writeError(w, http.StatusBadRequest, "no branch or tag "+body.Ref)
		return
	}
	file := ""
	for _, candidate := range workflowFiles(s.git, dir, sha) {
		if candidate == body.Workflow || path.Base(candidate) == body.Workflow {
			file = candidate
		}
	}
	if file == "" {
		writeError(w, http.StatusNotFound, "no workflow "+body.Workflow+" at "+body.Ref)
		return
	}
	content, err := s.git.Blob(dir, sha, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	workflow, err := model.ReadWorkflow(bytes.NewReader(content), false)
	if err != nil {
		writeError(w, http.StatusBadRequest, file+": "+err.Error())
		return
	}
	config := workflow.WorkflowDispatchConfig()
	if config == nil {
		writeError(w, http.StatusBadRequest, file+" has no workflow_dispatch trigger")
		return
	}
	inputs := map[string]any{}
	for name := range body.Inputs {
		if _, known := config.Inputs[name]; !known {
			writeError(w, http.StatusBadRequest, file+" has no input "+name)
			return
		}
	}
	for name, input := range config.Inputs {
		value, given := body.Inputs[name]
		if !given {
			value = input.Default
		}
		if !given && input.Required && input.Default == "" {
			writeError(w, http.StatusBadRequest, "input "+name+" is required")
			return
		}
		if input.Type == "boolean" && value != "" && value != "true" && value != "false" {
			writeError(w, http.StatusBadRequest, "input "+name+" must be true or false")
			return
		}
		if input.Type == "choice" && !slices.Contains(input.Options, value) {
			writeError(w, http.StatusBadRequest, "input "+name+" must be one of "+strings.Join(input.Options, ", "))
			return
		}
		inputs[name] = value
	}
	runs, messages := queueTrigger(s.store, s.git, dir, repo.Name, identityFrom(r).actor, trigger{
		event: "workflow_dispatch", ref: ref, sha: sha, file: file,
		payload: map[string]any{"inputs": inputs, "ref": ref, "workflow": file},
		matches: func(event *jobparser.Event) bool { return event.Name == "workflow_dispatch" },
		options: []jobparser.ParseOption{jobparser.WithInputs(inputs)},
	})
	if len(runs) == 0 {
		writeError(w, http.StatusBadRequest, strings.Join(messages, "; "))
		return
	}
	run, err := s.store.Run(repo.Name, runs[0].Number)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, withStepNames(run))
}

func matchesRelease(event *jobparser.Event, prerelease bool) bool {
	types, typed := event.Acts()["types"]
	if !typed {
		return true
	}
	return slices.Contains(types, "published") || slices.Contains(types, "released") && !prerelease || slices.Contains(types, "prereleased") && prerelease
}

func (s *Server) queueRelease(r *http.Request, release store.Release) []string {
	if release.Draft || release.Commit == "" {
		return nil
	}
	_, messages := queueTrigger(s.store, s.git, s.repoDir(release.Repo), release.Repo, identityFrom(r).actor, trigger{
		event: "release", ref: "refs/tags/" + release.Tag, sha: release.Commit,
		payload: map[string]any{
			"action": "published",
			"release": map[string]any{
				"id": release.ID, "tag_name": release.Tag, "name": release.Title, "body": release.Notes,
				"draft": release.Draft, "prerelease": release.Prerelease, "target_commitish": release.Commit,
			},
		},
		matches: func(event *jobparser.Event) bool {
			return event.Name == "release" && matchesRelease(event, release.Prerelease)
		},
	})
	return messages
}

func (s *Server) schedule(ctx context.Context) {
	for {
		next := time.Now().UTC().Truncate(time.Minute).Add(time.Minute)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
		}
		s.queueScheduled(next)
	}
}

func (s *Server) queueScheduled(minute time.Time) {
	if s.readOnly.Load() {
		return
	}
	s.gate.RLock()
	defer s.gate.RUnlock()
	repos, err := s.store.Repos()
	if err != nil {
		slog.Error("schedule", "error", err)
		return
	}
	actor := store.Actor{User: lyraUser, Via: "schedule"}
	for _, repo := range repos {
		dir := s.repoDir(repo.Name)
		branch := s.git.DefaultBranch(dir)
		sha, err := s.git.ResolveCommit(dir, "refs/heads/"+branch)
		if err != nil {
			continue
		}
		files := workflowFiles(s.git, dir, sha)
		sort.Strings(files)
		for _, file := range files {
			content, err := s.git.Blob(dir, sha, file)
			if err != nil {
				continue
			}
			workflow, err := model.ReadWorkflow(bytes.NewReader(content), false)
			if err != nil {
				continue
			}
			events, err := jobparser.ParseRawOn(&workflow.RawOn)
			if err != nil {
				continue
			}
			due := ""
			for _, event := range events {
				for _, entry := range event.Schedules() {
					parsed, err := cron.ParseStandard(entry["cron"])
					if err != nil {
						slog.Warn("schedule", "repo", repo.Name, "workflow", file, "cron", entry["cron"], "error", err)
						continue
					}
					if event.Name == "schedule" && parsed.Next(minute.Add(-time.Second)).Equal(minute) {
						due = entry["cron"]
					}
				}
			}
			if due == "" {
				continue
			}
			_, messages := queueTrigger(s.store, s.git, dir, repo.Name, actor, trigger{
				event: "schedule", ref: "refs/heads/" + branch, sha: sha, file: file,
				payload: map[string]any{"schedule": due},
				matches: func(event *jobparser.Event) bool { return event.Name == "schedule" },
			})
			slog.Info("schedule", "repo", repo.Name, "workflow", file, "cron", due, "result", strings.Join(messages, "; "))
		}
	}
}
