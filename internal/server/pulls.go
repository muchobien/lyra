package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

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
	Body      string   `json:"body"`
	Head      string   `json:"head"`
	Base      string   `json:"base"`
	Labels    []string `json:"labels"`
	Assignees []string `json:"assignees"`
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

func (s *Server) lookupPull(r *http.Request) (store.Repo, store.Pull, error) {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return repo, store.Pull{}, err
	}
	number, err := itemNumber(r)
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

func (s *Server) handleListPulls(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	query, err := parseItemQuery(r, pullStates)
	if err != nil {
		return err
	}
	pulls, err := s.store.Pulls(r.Context(), repo.Name, query.state)
	if err != nil {
		return err
	}
	matching := []store.Pull{}
	for _, pull := range pulls {
		if len(matching) < query.limit && query.matches(pull.Item) {
			matching = append(matching, pull)
		}
	}
	return writeJSON(w, http.StatusOK, matching)
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

func (s *Server) handleCreatePull(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return err
	}
	body, err := decodeBody[createPullRequest](w, r)
	if err != nil {
		return err
	}
	if err := checkText(&body.Title, &body.Body); err != nil {
		return err
	}
	dir := s.repoDir(repo.Name)
	branches, err := s.resolvePullBranches(ctx, dir, body.Head, body.Base)
	if err != nil {
		return err
	}
	id := identityFrom(r)
	pull, eventID, err := s.store.CreatePull(
		ctx,
		repo.Name,
		id.actor,
		body.Title,
		body.Body,
		branches.base,
		branches.head,
		branches.headSHA,
		branches.baseSHA,
		body.Labels,
		id.resolve(body.Assignees),
	)
	if err != nil {
		return missing(err, "repository not found")
	}
	if err := s.git.UpdateRef(ctx, dir, pullRef(pull.Number), branches.headSHA, ""); err != nil {
		return err
	}
	actions := QueuePullRuns(ctx, s.store, s.git, dir, repo.Name, id.actor, pull, "opened")
	if err := s.store.AnnounceChecks(ctx, repo.Name, pull.HeadSHA); err != nil {
		slog.Error("announce checks", "repo", repo.Name, "error", err)
	}
	return writeJSON(w, http.StatusCreated, pullWrite{Pull: pull, EventID: eventID, Actions: actions})
}

func (s *Server) handleGetPull(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, pull, err := s.lookupPull(r)
	if err != nil {
		return err
	}
	comments, err := s.store.Comments(ctx, pull.Item)
	if err != nil {
		return err
	}
	reviews, err := s.store.Reviews(ctx, pull.Item)
	if err != nil {
		return err
	}
	s.markOutdated(ctx, repo.Name, pull, comments)
	for index := range reviews {
		s.markOutdated(ctx, repo.Name, pull, reviews[index].Comments)
	}
	view := pullView{Pull: pull, Reviews: reviews, Comments: comments}
	if view.Checks, err = s.store.Checks(ctx, repo.Name, pull.HeadSHA); err != nil {
		return err
	}
	view.Merge, view.MergeError = s.mergeStatus(ctx, repo.Name, pull)
	return writeJSON(w, http.StatusOK, view)
}

func (s *Server) handlePullCommits(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, pull, err := s.lookupPull(r)
	if err != nil {
		return err
	}
	base, head := s.pullRange(ctx, repo.Name, pull)
	commits, err := s.commits(ctx, repo.Name, head, pullCommitLimit, base)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, commits)
}

func (s *Server) handlePullDiff(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, pull, err := s.lookupPull(r)
	if err != nil {
		return err
	}
	base, head := s.pullRange(ctx, repo.Name, pull)
	diff, err := s.git.Diff(ctx, s.repoDir(repo.Name), base, head)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/x-diff; charset=utf-8")
	w.Write([]byte(diff))
	return nil
}

func (s *Server) handlePullFiles(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, pull, err := s.lookupPull(r)
	if err != nil {
		return err
	}
	base, head := s.pullRange(ctx, repo.Name, pull)
	files, err := s.git.DiffFiles(ctx, s.repoDir(repo.Name), base, head)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, files)
}
