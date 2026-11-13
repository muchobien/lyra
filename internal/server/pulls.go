package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

var pullStates = []string{stateOpen, stateClosed, stateMerged, stateAll}

type pullView struct {
	store.Pull
	Merge      *mergePlan      `json:"merge,omitempty"`
	MergeError string          `json:"merge_error,omitempty"`
	Reviews    []store.Review  `json:"reviews"`
	Comments   []store.Comment `json:"comments"`
	Checks     store.Checks    `json:"checks"`
}

type createPullRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body,omitzero"`
	Head      string   `json:"head" doc:"The branch to merge."`
	Base      string   `json:"base,omitzero" doc:"Defaults to the default branch."`
	Labels    []string `json:"labels,omitzero"`
	Assignees []string `json:"assignees,omitzero"`
}

type createPullInput struct {
	RepoPath
	Body createPullRequest
}

type diffOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

type pullBranches struct {
	base    string
	head    string
	headSHA string
	baseSHA string
}

func pullRef(number int64) string {
	return fmt.Sprintf("refs/pull/%d/head", number)
}

func (s *Server) mergeStatus(ctx context.Context, repo string, pull store.Pull) (*mergePlan, string) {
	if pull.State != stateOpen {
		return nil, ""
	}
	plan, err := s.planMerge(ctx, repo, pull)
	if err != nil {
		return nil, err.Error()
	}
	return &plan, ""
}

func (s *Server) lookupPull(r *http.Request, number int64) (store.Repo, store.Pull, error) {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return repo, store.Pull{}, err
	}
	pull, err := s.store.Pull(r.Context(), repo.Name, number)
	return repo, pull, missing(err, "pull request not found")
}

func (s *Server) pullRange(ctx context.Context, repo string, pull store.Pull) (string, string) {
	if pull.State == stateOpen {
		base, err := s.git.MergeBase(ctx, s.repoDir(repo), "refs/heads/"+pull.Base, pull.HeadSHA)
		if err == nil {
			return base, pull.HeadSHA
		}
	}
	return pull.BaseSHA, pull.HeadSHA
}

func (s *Server) handleListPulls(r *http.Request, input *listItemsInput) (*body[[]store.Pull], error) {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	query, err := parseItemQuery(r, input, pullStates)
	if err != nil {
		return nil, err
	}
	pulls, err := s.store.Pulls(r.Context(), repo.Name, query.state)
	if err != nil {
		return nil, err
	}
	matching := []store.Pull{}
	for _, pull := range pulls {
		if len(matching) < query.limit && query.matches(pull.Item) {
			matching = append(matching, pull)
		}
	}
	return reply(matching)
}

func (s *Server) resolvePullBranches(ctx context.Context, dir, head, base string) (pullBranches, error) {
	branches := pullBranches{head: head, base: base}
	if branches.base == "" {
		branches.base = s.git.DefaultBranch(ctx, dir)
	}
	if head == "" || head == branches.base {
		return branches, badRequest("head must be a branch other than the base " + branches.base)
	}
	var err error
	if branches.headSHA, err = s.git.ResolveCommit(ctx, dir, "refs/heads/"+head); err != nil {
		return branches, badRequest("branch " + head + " does not exist")
	}
	baseTip, err := s.git.ResolveCommit(ctx, dir, "refs/heads/"+branches.base)
	if err != nil {
		return branches, badRequest("branch " + branches.base + " does not exist")
	}
	contained, err := s.git.IsAncestor(ctx, dir, branches.headSHA, baseTip)
	if err != nil {
		return branches, err
	}
	if contained {
		return branches, badRequest(branches.base + " already contains every commit on " + head)
	}
	if branches.baseSHA, err = s.git.MergeBase(ctx, dir, baseTip, branches.headSHA); err != nil {
		return branches, badRequest(head + " and " + branches.base + " share no history")
	}
	return branches, nil
}

func (s *Server) handleCreatePull(r *http.Request, input *createPullInput) (*body[pullWrite], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return nil, err
	}
	request := input.Body
	if err := checkText(&request.Title, &request.Body); err != nil {
		return nil, err
	}
	dir := s.repoDir(repo.Name)
	branches, err := s.resolvePullBranches(ctx, dir, request.Head, request.Base)
	if err != nil {
		return nil, err
	}
	id := identityFrom(r)
	pull, eventID, err := s.store.CreatePull(
		ctx,
		repo.Name,
		id.actor,
		request.Title,
		request.Body,
		branches.base,
		branches.head,
		branches.headSHA,
		branches.baseSHA,
		request.Labels,
		id.resolve(request.Assignees),
	)
	if err != nil {
		return nil, missing(err, "repository not found")
	}
	if err := s.git.UpdateRef(ctx, dir, pullRef(pull.Number), branches.headSHA, ""); err != nil {
		return nil, err
	}
	actions := QueuePullRuns(ctx, s.store, s.git, dir, repo.Name, id.actor, pull, "opened")
	if err := s.store.AnnounceChecks(ctx, repo.Name, pull.HeadSHA); err != nil {
		slog.Error("announce checks", "repo", repo.Name, "error", err)
	}
	return reply(pullWrite{Pull: pull, EventID: eventID, Actions: actions})
}

func (s *Server) handleGetPull(r *http.Request, input *ItemPath) (*body[pullView], error) {
	ctx := r.Context()
	repo, pull, err := s.lookupPull(r, input.Number)
	if err != nil {
		return nil, err
	}
	comments, err := s.store.Comments(ctx, pull.Item)
	if err != nil {
		return nil, err
	}
	reviews, err := s.store.Reviews(ctx, pull.Item)
	if err != nil {
		return nil, err
	}
	s.markOutdated(ctx, repo.Name, pull, comments)
	for index := range reviews {
		s.markOutdated(ctx, repo.Name, pull, reviews[index].Comments)
	}
	view := pullView{Pull: pull, Reviews: reviews, Comments: comments}
	if view.Checks, err = s.store.Checks(ctx, repo.Name, pull.HeadSHA); err != nil {
		return nil, err
	}
	view.Merge, view.MergeError = s.mergeStatus(ctx, repo.Name, pull)
	return reply(view)
}

func (s *Server) handleEditPull(r *http.Request, input *editItemInput) (*body[pullWrite], error) {
	edited, err := s.editItem(r, kindPull, input)
	if err != nil {
		return nil, err
	}
	pull, err := s.store.Pull(r.Context(), edited.repo, input.Number)
	if err != nil {
		return nil, err
	}
	return reply(pullWrite{Pull: pull, EventID: edited.eventID, Actions: edited.actions})
}

func (s *Server) handlePullCommits(r *http.Request, input *ItemPath) (*body[[]repos.Commit], error) {
	ctx := r.Context()
	repo, pull, err := s.lookupPull(r, input.Number)
	if err != nil {
		return nil, err
	}
	base, head := s.pullRange(ctx, repo.Name, pull)
	commits, err := s.commits(ctx, repo.Name, head, pullCommitLimit, base)
	if err != nil {
		return nil, err
	}
	return reply(commits)
}

func (s *Server) handlePullDiff(r *http.Request, input *ItemPath) (*diffOutput, error) {
	ctx := r.Context()
	repo, pull, err := s.lookupPull(r, input.Number)
	if err != nil {
		return nil, err
	}
	base, head := s.pullRange(ctx, repo.Name, pull)
	diff, err := s.git.Diff(ctx, s.repoDir(repo.Name), base, head)
	if err != nil {
		return nil, err
	}
	return &diffOutput{ContentType: "text/x-diff; charset=utf-8", Body: []byte(diff)}, nil
}

func (s *Server) handlePullFiles(r *http.Request, input *ItemPath) (*body[[]repos.FileChange], error) {
	ctx := r.Context()
	repo, pull, err := s.lookupPull(r, input.Number)
	if err != nil {
		return nil, err
	}
	base, head := s.pullRange(ctx, repo.Name, pull)
	files, err := s.git.DiffFiles(ctx, s.repoDir(repo.Name), base, head)
	if err != nil {
		return nil, err
	}
	return reply(files)
}
