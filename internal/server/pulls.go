package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/muchobien/lyra/internal/store"
)

type pullView struct {
	store.Pull
	Merge      *mergePlan      `json:"merge,omitempty"`
	MergeError string          `json:"merge_error,omitempty"`
	Reviews    []store.Review  `json:"reviews"`
	Comments   []store.Comment `json:"comments"`
	Checks     store.Checks    `json:"checks"`
}

func (s *Server) mergeStatus(repo string, pull store.Pull) (*mergePlan, string) {
	if pull.State != "open" {
		return nil, ""
	}
	plan, err := s.planMerge(repo, pull)
	if err != nil {
		return nil, err.Error()
	}
	return &plan, ""
}

func (s *Server) lookupPull(w http.ResponseWriter, r *http.Request) (store.Repo, store.Pull, bool) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return repo, store.Pull{}, false
	}
	number, ok := itemNumber(w, r)
	if !ok {
		return repo, store.Pull{}, false
	}
	pull, err := s.store.Pull(repo.Name, number)
	if err != nil {
		writeStoreError(w, err, "pull request not found")
		return repo, pull, false
	}
	return repo, pull, true
}

func (s *Server) pullRange(repo string, pull store.Pull) (string, string) {
	if pull.State == "open" {
		if base, err := s.git.MergeBase(s.repoDir(repo), "refs/heads/"+pull.Base, pull.HeadSHA); err == nil {
			return base, pull.HeadSHA
		}
	}
	return pull.BaseSHA, pull.HeadSHA
}

func (s *Server) handleListPulls(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	query, ok := parseItemQuery(w, r, []string{"open", "closed", "merged", "all"})
	if !ok {
		return
	}
	pulls, err := s.store.Pulls(repo.Name, query.state)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	matching := []store.Pull{}
	for _, pull := range pulls {
		if len(matching) < query.limit && query.matches(pull.Item) {
			matching = append(matching, pull)
		}
	}
	writeJSON(w, http.StatusOK, matching)
}

func (s *Server) handleCreatePull(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	var body struct {
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		Head      string   `json:"head"`
		Base      string   `json:"base"`
		Labels    []string `json:"labels"`
		Assignees []string `json:"assignees"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if problem := checkText(&body.Title, &body.Body); problem != "" {
		writeError(w, http.StatusBadRequest, problem)
		return
	}
	dir := s.repoDir(repo.Name)
	if body.Base == "" {
		body.Base = s.git.DefaultBranch(dir)
	}
	if body.Head == "" || body.Head == body.Base {
		writeError(w, http.StatusBadRequest, "head must be a branch other than the base "+body.Base)
		return
	}
	headSHA, err := s.git.ResolveCommit(dir, "refs/heads/"+body.Head)
	if err != nil {
		writeError(w, http.StatusBadRequest, "branch "+body.Head+" does not exist")
		return
	}
	baseTip, err := s.git.ResolveCommit(dir, "refs/heads/"+body.Base)
	if err != nil {
		writeError(w, http.StatusBadRequest, "branch "+body.Base+" does not exist")
		return
	}
	contained, err := s.git.IsAncestor(dir, headSHA, baseTip)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if contained {
		writeError(w, http.StatusBadRequest, body.Base+" already contains every commit on "+body.Head)
		return
	}
	baseSHA, err := s.git.MergeBase(dir, baseTip, headSHA)
	if err != nil {
		writeError(w, http.StatusBadRequest, body.Head+" and "+body.Base+" share no history")
		return
	}
	id := identityFrom(r)
	pull, eventID, err := s.store.CreatePull(repo.Name, id.actor, body.Title, body.Body, body.Base, body.Head, headSHA, baseSHA, body.Labels, id.resolve(body.Assignees))
	if err != nil {
		writeStoreError(w, err, "repository not found")
		return
	}
	if err := s.git.UpdateRef(dir, fmt.Sprintf("refs/pull/%d/head", pull.Number), headSHA, ""); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	actions := QueuePullRuns(s.store, s.git, dir, repo.Name, id.actor, pull, "opened")
	if err := s.store.AnnounceChecks(repo.Name, pull.HeadSHA); err != nil {
		slog.Error("announce checks", "repo", repo.Name, "error", err)
	}
	writeJSON(w, http.StatusCreated, pullWrite{Pull: pull, EventID: eventID, Actions: actions})
}

func (s *Server) handleGetPull(w http.ResponseWriter, r *http.Request) {
	repo, pull, ok := s.lookupPull(w, r)
	if !ok {
		return
	}
	comments, err := s.store.Comments(pull.Item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	reviews, err := s.store.Reviews(pull.Item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.markOutdated(repo.Name, pull, comments)
	for index := range reviews {
		s.markOutdated(repo.Name, pull, reviews[index].Comments)
	}
	view := pullView{Pull: pull, Reviews: reviews, Comments: comments}
	if view.Checks, err = s.store.Checks(repo.Name, pull.HeadSHA); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	view.Merge, view.MergeError = s.mergeStatus(repo.Name, pull)
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handlePullCommits(w http.ResponseWriter, r *http.Request) {
	repo, pull, ok := s.lookupPull(w, r)
	if !ok {
		return
	}
	base, head := s.pullRange(repo.Name, pull)
	commits, err := s.commits(repo.Name, head, 250, base)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, commits)
}

func (s *Server) handlePullDiff(w http.ResponseWriter, r *http.Request) {
	repo, pull, ok := s.lookupPull(w, r)
	if !ok {
		return
	}
	base, head := s.pullRange(repo.Name, pull)
	diff, err := s.git.Diff(s.repoDir(repo.Name), base, head)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/x-diff; charset=utf-8")
	w.Write([]byte(diff))
}

func (s *Server) handlePullFiles(w http.ResponseWriter, r *http.Request) {
	repo, pull, ok := s.lookupPull(w, r)
	if !ok {
		return
	}
	base, head := s.pullRange(repo.Name, pull)
	files, err := s.git.DiffFiles(s.repoDir(repo.Name), base, head)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, files)
}
