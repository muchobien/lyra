package server

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"path"
	"slices"
	"strings"

	"code.forgejo.org/forgejo/runner/v13/act/jobparser"
	"code.forgejo.org/forgejo/runner/v13/act/model"
	"code.forgejo.org/forgejo/runner/v13/act/workflowpattern"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

const (
	eventPush        = "push"
	eventPullRequest = "pull_request"
	eventDispatch    = "workflow_dispatch"
	eventRelease     = "release"
	eventSchedule    = "schedule"
)

var workflowDirs = []string{".forgejo/workflows", ".github/workflows"}

var defaultPullTypes = []string{"opened", "synchronize", "reopened"}

type loginPayload struct {
	Login    string `json:"login"`
	Username string `json:"username"`
}

type repositoryPayload struct {
	DefaultBranch string       `json:"default_branch"`
	FullName      string       `json:"full_name"`
	Name          string       `json:"name"`
	Owner         loginPayload `json:"owner"`
}

type headCommitPayload struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type pusherPayload struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

type repoNamePayload struct {
	FullName string `json:"full_name"`
}

type branchPayload struct {
	Ref  string          `json:"ref"`
	Repo repoNamePayload `json:"repo"`
	SHA  string          `json:"sha"`
}

type pullRequestPayload struct {
	Base   branchPayload `json:"base"`
	Body   string        `json:"body"`
	Head   branchPayload `json:"head"`
	Number int64         `json:"number"`
	State  string        `json:"state"`
	Title  string        `json:"title"`
}

type releasePayload struct {
	Body            string `json:"body"`
	Draft           bool   `json:"draft"`
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Prerelease      bool   `json:"prerelease"`
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
}

type workflowEvent struct {
	Action      string              `json:"action,omitempty"`
	After       string              `json:"after,omitempty"`
	Before      string              `json:"before,omitempty"`
	HeadCommit  *headCommitPayload  `json:"head_commit,omitempty"`
	Inputs      map[string]any      `json:"inputs,omitzero"`
	Number      int64               `json:"number,omitzero"`
	PullRequest *pullRequestPayload `json:"pull_request,omitempty"`
	Pusher      *pusherPayload      `json:"pusher,omitempty"`
	Ref         string              `json:"ref,omitempty"`
	Release     *releasePayload     `json:"release,omitempty"`
	Repository  repositoryPayload   `json:"repository"`
	Schedule    string              `json:"schedule,omitempty"`
	Sender      loginPayload        `json:"sender"`
	Workflow    string              `json:"workflow,omitempty"`
}

type trigger struct {
	event   string
	ref     string
	sha     string
	pull    int64
	headRef string
	baseRef string
	payload workflowEvent
	matches func(*jobparser.Event) bool
	file    string
	options []jobparser.ParseOption
}

type workflowQueue struct {
	db    *store.Store
	git   repos.Git
	dir   string
	repo  string
	actor store.Actor
}

func isWorkflowFile(entry repos.Entry) bool {
	return entry.Type == "blob" && (strings.HasSuffix(entry.Name, ".yml") || strings.HasSuffix(entry.Name, ".yaml"))
}

func workflowFiles(ctx context.Context, git repos.Git, dir, sha string) []string {
	for _, workflowDir := range workflowDirs {
		entries, err := git.Tree(ctx, dir, sha, workflowDir)
		if err != nil {
			continue
		}
		files := []string{}
		for _, entry := range entries {
			if isWorkflowFile(entry) {
				files = append(files, workflowDir+"/"+entry.Name)
			}
		}
		return files
	}
	return nil
}

func hasAny(acts map[string][]string, names ...string) bool {
	return slices.ContainsFunc(names, func(name string) bool {
		_, present := acts[name]
		return present
	})
}

func matchesPush(event *jobparser.Event, ref string, changed []string) bool {
	branch, isBranch := strings.CutPrefix(ref, "refs/heads/")
	tag, isTag := strings.CutPrefix(ref, "refs/tags/")
	acts := event.Acts()
	branches := hasAny(acts, "branches", "branches-ignore")
	tags := hasAny(acts, "tags", "tags-ignore")
	if (branches && !tags && !isBranch) || (tags && !branches && !isTag) {
		return false
	}
	trace := &workflowpattern.EmptyTraceWriter{}
	for name, patterns := range acts {
		compiled, err := workflowpattern.CompilePatterns(patterns...)
		if err != nil {
			return false
		}
		switch {
		case name == "branches" && isBranch && workflowpattern.Skip(compiled, []string{branch}, trace):
			return false
		case name == "branches-ignore" && isBranch && workflowpattern.Filter(compiled, []string{branch}, trace):
			return false
		case name == "tags" && isTag && workflowpattern.Skip(compiled, []string{tag}, trace):
			return false
		case name == "tags-ignore" && isTag && workflowpattern.Filter(compiled, []string{tag}, trace):
			return false
		case name == "paths" && changed != nil && workflowpattern.Skip(compiled, changed, trace):
			return false
		case name == "paths-ignore" && changed != nil && workflowpattern.Filter(compiled, changed, trace):
			return false
		}
	}
	return true
}

func matchesPull(event *jobparser.Event, action, base string, changed []string) bool {
	acts := event.Acts()
	types, typed := acts["types"]
	if !typed {
		types = defaultPullTypes
	}
	if !slices.Contains(types, action) {
		return false
	}
	trace := &workflowpattern.EmptyTraceWriter{}
	for name, patterns := range acts {
		compiled, err := workflowpattern.CompilePatterns(patterns...)
		if err != nil {
			return false
		}
		switch {
		case name == "branches" && workflowpattern.Skip(compiled, []string{base}, trace):
			return false
		case name == "branches-ignore" && workflowpattern.Filter(compiled, []string{base}, trace):
			return false
		case name == "paths" && workflowpattern.Skip(compiled, changed, trace):
			return false
		case name == "paths-ignore" && workflowpattern.Filter(compiled, changed, trace):
			return false
		}
	}
	return true
}

func changedFiles(ctx context.Context, git repos.Git, dir, from, to string) []string {
	files, err := git.DiffFiles(ctx, dir, from, to)
	if err != nil {
		return nil
	}
	changed := []string{}
	for _, file := range files {
		changed = append(changed, file.Path)
	}
	return changed
}

func workflowJobs(content []byte, options []jobparser.ParseOption) ([]store.Job, error) {
	singles, err := jobparser.Parse(content, false, options...)
	if err != nil {
		return nil, err
	}
	jobs := []store.Job{}
	for _, single := range singles {
		key, job := single.Job()
		encoded, err := single.Marshal()
		if err != nil {
			return nil, err
		}
		name := job.Name
		if name == "" {
			name = key
		}
		jobs = append(jobs, store.Job{
			Key:         key,
			Name:        name,
			Needs:       job.Needs(),
			RunsOn:      job.RunsOn(),
			Conditional: job.If.Value != "",
			Payload:     string(encoded),
		})
	}
	return jobs, nil
}

func (queue workflowQueue) parse(ctx context.Context, cause trigger, file string) (string, []store.Job, error) {
	content, err := queue.git.Blob(ctx, queue.dir, cause.sha, file)
	if err != nil {
		return "", nil, err
	}
	workflow, err := model.ReadWorkflow(bytes.NewReader(content), false)
	if err != nil {
		return "", nil, err
	}
	events, err := jobparser.ParseRawOn(&workflow.RawOn)
	if err != nil {
		return "", nil, err
	}
	if !slices.ContainsFunc(events, cause.matches) {
		return "", nil, nil
	}
	name := workflow.Name
	if name == "" {
		name = path.Base(file)
	}
	jobs, err := workflowJobs(content, cause.options)
	return name, jobs, err
}

func (queue workflowQueue) trigger(ctx context.Context, cause trigger) ([]store.Run, []string) {
	runs, messages := []store.Run{}, []string{}
	commit, err := queue.git.Commit(ctx, queue.dir, cause.sha)
	if err != nil {
		return runs, append(messages, "actions: "+err.Error())
	}
	owner, name, _ := strings.Cut(queue.repo, "/")
	cause.payload.Repository = repositoryPayload{
		DefaultBranch: queue.git.DefaultBranch(ctx, queue.dir),
		FullName:      queue.repo,
		Name:          name,
		Owner:         loginPayload{Login: owner, Username: owner},
	}
	cause.payload.Sender = loginPayload{Login: queue.actor.User, Username: queue.actor.User}
	payload, err := json.Marshal(cause.payload, json.Deterministic(true))
	if err != nil {
		return runs, append(messages, "actions: "+err.Error())
	}
	for _, file := range workflowFiles(ctx, queue.git, queue.dir, cause.sha) {
		if cause.file != "" && file != cause.file {
			continue
		}
		runName, jobs, err := queue.parse(ctx, cause, file)
		if err != nil {
			messages = append(messages, file+": "+err.Error())
			continue
		}
		if jobs == nil {
			continue
		}
		run, _, err := queue.db.CreateRun(ctx, queue.actor, store.Run{
			Repo:     queue.repo,
			Workflow: file,
			Name:     runName,
			Title:    commit.Subject,
			Event:    cause.event,
			Ref:      cause.ref,
			SHA:      cause.sha,
			Payload:  string(payload),
			Pull:     cause.pull,
			HeadRef:  cause.headRef,
			BaseRef:  cause.baseRef,
		}, jobs)
		if err != nil {
			messages = append(messages, file+": "+err.Error())
			continue
		}
		messages = append(messages, fmt.Sprintf("queued run #%d (%s)", run.Number, runName))
		runs = append(runs, run)
	}
	return runs, messages
}

func isBranchOrTag(ref string) bool {
	return strings.HasPrefix(ref, "refs/heads/") || strings.HasPrefix(ref, "refs/tags/")
}

func (queue workflowQueue) push(ctx context.Context, update RefUpdate) []string {
	var changed []string
	if !zeroSHA(update.Old) {
		changed = changedFiles(ctx, queue.git, queue.dir, update.Old, update.New)
	}
	commit, err := queue.git.Commit(ctx, queue.dir, update.New)
	if err != nil {
		return []string{"actions: " + err.Error()}
	}
	_, queued := queue.trigger(ctx, trigger{
		event: eventPush,
		ref:   update.Ref,
		sha:   update.New,
		payload: workflowEvent{
			After:  update.New,
			Before: update.Old,
			HeadCommit: &headCommitPayload{
				ID:        commit.SHA,
				Message:   strings.TrimSpace(commit.Subject + "\n\n" + commit.Body),
				Timestamp: commit.Committer.Date,
			},
			Pusher: &pusherPayload{Name: queue.actor.User, Username: queue.actor.User},
			Ref:    update.Ref,
		},
		matches: func(event *jobparser.Event) bool {
			return event.Name == eventPush && matchesPush(event, update.Ref, changed)
		},
	})
	return queued
}

func QueueRuns(
	ctx context.Context,
	db *store.Store,
	git repos.Git,
	dir, repo string,
	actor store.Actor,
	updates []RefUpdate,
) []string {
	queue := workflowQueue{db: db, git: git, dir: dir, repo: repo, actor: actor}
	messages := []string{}
	for _, update := range updates {
		if zeroSHA(update.New) || !isBranchOrTag(update.Ref) {
			continue
		}
		messages = append(messages, queue.push(ctx, update)...)
	}
	return messages
}

func QueuePullRuns(
	ctx context.Context,
	db *store.Store,
	git repos.Git,
	dir, repo string,
	actor store.Actor,
	pull store.Pull,
	action string,
) []string {
	queue := workflowQueue{db: db, git: git, dir: dir, repo: repo, actor: actor}
	changed := changedFiles(ctx, git, dir, pull.BaseSHA, pull.HeadSHA)
	source := repoNamePayload{FullName: repo}
	_, messages := queue.trigger(ctx, trigger{
		event:   eventPullRequest,
		ref:     pullRef(pull.Number),
		sha:     pull.HeadSHA,
		pull:    pull.Number,
		headRef: pull.Head,
		baseRef: pull.Base,
		payload: workflowEvent{
			Action: action,
			Number: pull.Number,
			PullRequest: &pullRequestPayload{
				Base:   branchPayload{Ref: pull.Base, Repo: source, SHA: pull.BaseSHA},
				Body:   pull.Body,
				Head:   branchPayload{Ref: pull.Head, Repo: source, SHA: pull.HeadSHA},
				Number: pull.Number,
				State:  pull.State,
				Title:  pull.Title,
			},
		},
		matches: func(event *jobparser.Event) bool {
			return event.Name == eventPullRequest && matchesPull(event, action, pull.Base, changed)
		},
	})
	return messages
}

func (s *Server) queue(repo string, actor store.Actor) workflowQueue {
	return workflowQueue{db: s.store, git: s.git, dir: s.repoDir(repo), repo: repo, actor: actor}
}

func (s *Server) syncReopened(ctx context.Context, pull *store.Pull, actor store.Actor) error {
	dir := s.repoDir(pull.Repo)
	tip, err := s.git.ResolveCommit(ctx, dir, "refs/heads/"+pull.Head)
	if err != nil || tip == pull.HeadSHA {
		return nil
	}
	base, err := s.git.MergeBase(ctx, dir, "refs/heads/"+pull.Base, tip)
	if err != nil {
		base = pull.BaseSHA
	}
	if err := s.git.UpdateRef(ctx, dir, pullRef(pull.Number), tip, ""); err != nil {
		return err
	}
	if _, err := s.store.SyncPull(ctx, pull.Repo, pull.Number, actor, tip, base); err != nil {
		return err
	}
	pull.HeadSHA, pull.BaseSHA = tip, base
	return nil
}

func (s *Server) queueReopened(
	ctx context.Context,
	repo string,
	number int64,
	wasOpen bool,
	actor store.Actor,
) []string {
	if wasOpen {
		return nil
	}
	pull, err := s.store.Pull(ctx, repo, number)
	if err != nil || pull.State != stateOpen {
		return nil
	}
	if err := s.syncReopened(ctx, &pull, actor); err != nil {
		return []string{"sync: " + err.Error()}
	}
	actions := []string{}
	if !s.actionsRepo(repo) {
		actions = QueuePullRuns(ctx, s.store, s.git, s.repoDir(repo), repo, actor, pull, "reopened")
	}
	if err := s.store.AnnounceChecks(ctx, repo, pull.HeadSHA); err != nil {
		slog.Error("announce checks", "repo", repo, "error", err)
	}
	return actions
}

func (s *Server) checkReopen(ctx context.Context, repo string, number int64, state *string) (bool, error) {
	pull, err := s.store.Pull(ctx, repo, number)
	if err != nil {
		return true, nil
	}
	reopening := pull.State == stateClosed && state != nil && *state == stateOpen
	if !reopening {
		return pull.State == stateOpen, nil
	}
	if _, err := s.git.ResolveCommit(ctx, s.repoDir(repo), "refs/heads/"+pull.Head); err != nil {
		return false, conflict("branch " + pull.Head + " no longer exists; push it again to reopen this pull request")
	}
	return false, nil
}
