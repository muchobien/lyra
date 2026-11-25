package server

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/muchobien/lyra/internal/repos"
)

type branchView struct {
	Name      string       `json:"name"`
	Default   bool         `json:"default"`
	Protected bool         `json:"protected" doc:"A branch rule forbids deleting it."`
	Commit    repos.Commit `json:"commit"`
	Ahead     int          `json:"ahead" doc:"Commits on the branch that aren't on the default branch."`
	Behind    int          `json:"behind" doc:"Commits on the default branch that aren't on the branch."`
}

type deleteBranchInput struct {
	RepoPath
	Name string `query:"name" required:"true" doc:"The branch name, such as feature/login."`
}

func (s *Server) handleListBranches(r *http.Request, input *RepoPath) (*body[[]branchView], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead, input.fullName())
	if err != nil {
		return nil, err
	}
	dir := s.repoDir(repo.Name)
	branches, err := s.git.Branches(ctx, dir)
	if err != nil {
		return nil, err
	}
	rules, err := s.store.Rules(ctx, repo.Name)
	if err != nil {
		return nil, err
	}
	defaultBranch := s.git.DefaultBranch(ctx, dir)
	views := []branchView{}
	for _, branch := range branches {
		commits, err := s.commits(ctx, repo.Name, branch.SHA, 1)
		if err != nil {
			return nil, err
		}
		view := branchView{
			Name: branch.Name, Default: branch.Name == defaultBranch, Commit: commits[0],
			Protected: policyFor(rules, branch.Name).forbidDeletion,
		}
		if !view.Default {
			ahead, behind, err := s.git.AheadBehind(ctx, dir, "refs/heads/"+defaultBranch, branch.SHA)
			if err == nil {
				view.Ahead, view.Behind = ahead, behind
			}
		}
		views = append(views, view)
	}
	slices.SortStableFunc(views, func(first, second branchView) int {
		if first.Default != second.Default {
			if first.Default {
				return -1
			}
			return 1
		}
		firstDate, _ := time.Parse(time.RFC3339, first.Commit.Committer.Date)
		secondDate, _ := time.Parse(time.RFC3339, second.Commit.Committer.Date)
		return secondDate.Compare(firstDate)
	})
	return reply(views)
}

func (s *Server) handleDeleteBranch(r *http.Request, input *deleteBranchInput) (*body[deletedWithEvent], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return nil, err
	}
	dir := s.repoDir(repo.Name)
	if input.Name == s.git.DefaultBranch(ctx, dir) {
		return nil, conflict(input.Name + " is the default branch")
	}
	ref := "refs/heads/" + input.Name
	sha, err := s.git.ResolveCommit(ctx, dir, ref)
	if err != nil {
		return nil, notFound("branch " + input.Name + " not found")
	}
	updates := []RefUpdate{{Ref: ref, Old: sha, New: strings.Repeat("0", len(sha))}}
	if err := CheckPush(ctx, s.store, s.git, dir, repo.Name, updates); err != nil {
		return nil, conflict(err.Error())
	}
	if err := s.git.UpdateRef(ctx, dir, ref, "", sha); err != nil {
		return nil, conflict(input.Name + " moved while deleting it; try again")
	}
	actor := identityFrom(r).actor
	eventID, err := s.store.AddEvent(ctx, "push", repo.Name, actor, PushEvent{Refs: updates})
	if err != nil {
		return nil, err
	}
	queueRuns := !s.actionsRepo(repo.Name)
	if _, err := ProcessPush(ctx, s.store, s.git, dir, repo.Name, actor, updates, queueRuns); err != nil {
		return nil, err
	}
	return reply(deletedWithEvent{Deleted: input.Name, EventID: eventID})
}
