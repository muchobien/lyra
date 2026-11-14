package server

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"code.forgejo.org/forgejo/runner/v13/act/jobparser"
	"code.forgejo.org/forgejo/runner/v13/act/model"
	"github.com/robfig/cron/v3"

	"github.com/muchobien/lyra/internal/store"
)

type dispatchRequest struct {
	Workflow string            `json:"workflow" doc:"The workflow file name."`
	Ref      string            `json:"ref,omitzero" doc:"Defaults to the default branch."`
	Inputs   map[string]string `json:"inputs,omitzero"`
}

func (s *Server) dispatchRef(ctx context.Context, dir, name string) (string, string, error) {
	for _, candidate := range []string{"refs/heads/" + name, "refs/tags/" + name, name} {
		if !strings.HasPrefix(candidate, "refs/") {
			continue
		}
		if sha, err := s.git.ResolveCommit(ctx, dir, candidate); err == nil {
			return candidate, sha, nil
		}
	}
	return "", "", badRequest("no branch or tag " + name)
}

func (s *Server) dispatchWorkflow(
	ctx context.Context,
	dir, sha string,
	body dispatchRequest,
) (string, *model.WorkflowDispatch, error) {
	file := ""
	for _, candidate := range workflowFiles(ctx, s.git, dir, sha) {
		if candidate == body.Workflow || path.Base(candidate) == body.Workflow {
			file = candidate
		}
	}
	if file == "" {
		return "", nil, notFound("no workflow " + body.Workflow + " at " + body.Ref)
	}
	content, err := s.git.Blob(ctx, dir, sha, file)
	if err != nil {
		return file, nil, err
	}
	workflow, err := model.ReadWorkflow(bytes.NewReader(content), false)
	if err != nil {
		return file, nil, badRequest(file + ": " + err.Error())
	}
	config := workflow.WorkflowDispatchConfig()
	if config == nil {
		return file, nil, badRequest(file + " has no workflow_dispatch trigger")
	}
	return file, config, nil
}

func dispatchInput(name string, input model.WorkflowDispatchInput, given map[string]string) (string, error) {
	value, present := given[name]
	if !present {
		value = input.Default
	}
	if !present && input.Required && input.Default == "" {
		return "", badRequest("input " + name + " is required")
	}
	if input.Type == "boolean" && value != "" && value != "true" && value != "false" {
		return "", badRequest("input " + name + " must be true or false")
	}
	if input.Type == "choice" && !slices.Contains(input.Options, value) {
		return "", badRequest("input " + name + " must be one of " + strings.Join(input.Options, ", "))
	}
	return value, nil
}

func dispatchInputs(file string, config *model.WorkflowDispatch, given map[string]string) (map[string]any, error) {
	for name := range given {
		if _, known := config.Inputs[name]; !known {
			return nil, badRequest(file + " has no input " + name)
		}
	}
	inputs := map[string]any{}
	for name, input := range config.Inputs {
		value, err := dispatchInput(name, input, given)
		if err != nil {
			return nil, err
		}
		inputs[name] = value
	}
	return inputs, nil
}

type dispatchWorkflowInput struct {
	RepoPath
	Body dispatchRequest
}

func (s *Server) handleDispatch(r *http.Request, input *dispatchWorkflowInput) (*body[store.Run], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return nil, err
	}
	if s.actionsRepo(repo.Name) {
		return nil, badRequest("repositories of actions owners run no workflows")
	}
	request := input.Body
	dir := s.repoDir(repo.Name)
	if request.Ref == "" {
		request.Ref = s.git.DefaultBranch(ctx, dir)
	}
	ref, sha, err := s.dispatchRef(ctx, dir, request.Ref)
	if err != nil {
		return nil, err
	}
	file, config, err := s.dispatchWorkflow(ctx, dir, sha, request)
	if err != nil {
		return nil, err
	}
	inputs, err := dispatchInputs(file, config, request.Inputs)
	if err != nil {
		return nil, err
	}
	runs, messages := s.queue(repo.Name, identityFrom(r).actor).trigger(ctx, trigger{
		event:   eventDispatch,
		ref:     ref,
		sha:     sha,
		file:    file,
		payload: workflowEvent{Inputs: inputs, Ref: ref, Workflow: file},
		matches: func(event *jobparser.Event) bool { return event.Name == eventDispatch },
		options: []jobparser.ParseOption{jobparser.WithInputs(inputs)},
	})
	if len(runs) == 0 {
		return nil, badRequest(strings.Join(messages, "; "))
	}
	run, err := s.store.Run(ctx, repo.Name, runs[0].Number)
	if err != nil {
		return nil, err
	}
	return reply(withStepNames(run))
}

func matchesRelease(event *jobparser.Event, prerelease bool) bool {
	types, typed := event.Acts()["types"]
	if !typed || slices.Contains(types, "published") {
		return true
	}
	if prerelease {
		return slices.Contains(types, "prereleased")
	}
	return slices.Contains(types, "released")
}

func (s *Server) queueRelease(r *http.Request, release store.Release) []string {
	if release.Draft || release.Commit == "" || s.actionsRepo(release.Repo) {
		return nil
	}
	_, messages := s.queue(release.Repo, identityFrom(r).actor).trigger(r.Context(), trigger{
		event: eventRelease,
		ref:   tagRef(release.Tag),
		sha:   release.Commit,
		payload: workflowEvent{
			Action: "published",
			Release: &releasePayload{
				Body:            release.Notes,
				Draft:           release.Draft,
				ID:              release.ID,
				Name:            release.Title,
				Prerelease:      release.Prerelease,
				TagName:         release.Tag,
				TargetCommitish: release.Commit,
			},
		},
		matches: func(event *jobparser.Event) bool {
			return event.Name == eventRelease && matchesRelease(event, release.Prerelease)
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
		s.queueScheduled(ctx, next)
	}
}

func dueCron(repo, file string, events []*jobparser.Event, minute time.Time) string {
	due := ""
	for _, event := range events {
		for _, entry := range event.Schedules() {
			parsed, err := cron.ParseStandard(entry["cron"])
			if err != nil {
				slog.Warn("schedule", "repo", repo, "workflow", file, "cron", entry["cron"], "error", err)
				continue
			}
			if event.Name == eventSchedule && parsed.Next(minute.Add(-time.Second)).Equal(minute) {
				due = entry["cron"]
			}
		}
	}
	return due
}

func (s *Server) scheduledCron(ctx context.Context, repo, sha, file string, minute time.Time) string {
	content, err := s.git.Blob(ctx, s.repoDir(repo), sha, file)
	if err != nil {
		return ""
	}
	workflow, err := model.ReadWorkflow(bytes.NewReader(content), false)
	if err != nil {
		return ""
	}
	events, err := jobparser.ParseRawOn(&workflow.RawOn)
	if err != nil {
		return ""
	}
	return dueCron(repo, file, events, minute)
}

func (s *Server) queueRepoSchedules(ctx context.Context, repo string, minute time.Time) {
	dir := s.repoDir(repo)
	branch := s.git.DefaultBranch(ctx, dir)
	sha, err := s.git.ResolveCommit(ctx, dir, "refs/heads/"+branch)
	if err != nil {
		return
	}
	queue := s.queue(repo, store.Actor{User: lyraUser, Via: "schedule"})
	files := workflowFiles(ctx, s.git, dir, sha)
	slices.Sort(files)
	for _, file := range files {
		due := s.scheduledCron(ctx, repo, sha, file, minute)
		if due == "" {
			continue
		}
		_, messages := queue.trigger(ctx, trigger{
			event:   eventSchedule,
			ref:     "refs/heads/" + branch,
			sha:     sha,
			file:    file,
			payload: workflowEvent{Schedule: due},
			matches: func(event *jobparser.Event) bool { return event.Name == eventSchedule },
		})
		slog.Info("schedule", "repo", repo, "workflow", file, "cron", due, "result", strings.Join(messages, "; "))
	}
}

func (s *Server) queueScheduled(ctx context.Context, minute time.Time) {
	if s.readOnly.Load() {
		return
	}
	s.gate.RLock()
	defer s.gate.RUnlock()
	repos, err := s.store.Repos(ctx)
	if err != nil {
		slog.Error("schedule", "error", err)
		return
	}
	for _, repo := range repos {
		if !s.actionsRepo(repo.Name) {
			s.queueRepoSchedules(ctx, repo.Name, minute)
		}
	}
}
