package server

import (
	"net/http"

	"github.com/muchobien/lyra/internal/repos"
)

type compareInput struct {
	RepoPath
	Base string `query:"base" doc:"Branch, tag, or commit to compare against; defaults to the default branch."`
	Head string `query:"head" required:"true" doc:"Branch, tag, or commit with the changes."`
}

type compareRange struct {
	repo      string
	baseSHA   string
	headSHA   string
	mergeBase string
	from      string
}

type compareView struct {
	BaseSHA   string             `json:"base_sha"`
	HeadSHA   string             `json:"head_sha"`
	MergeBase string             `json:"merge_base" doc:"Empty when the two refs share no history."`
	Commits   []repos.Commit     `json:"commits" doc:"Commits on head that aren't on base, newest first, at most 250."`
	Files     []repos.FileChange `json:"files" doc:"Files head changed since the merge base, or base without one."`
}

func (s *Server) resolveCompare(r *http.Request, input *compareInput) (compareRange, error) {
	repo, err := s.lookupRepo(r, scopeRead, input.fullName())
	if err != nil {
		return compareRange{}, err
	}
	ctx := r.Context()
	dir := s.repoDir(repo.Name)
	base := input.Base
	if base == "" {
		base = s.git.DefaultBranch(ctx, dir)
	}
	result := compareRange{repo: repo.Name}
	if result.baseSHA, err = s.git.ResolveCommit(ctx, dir, base); err != nil {
		return compareRange{}, notFound("unknown ref " + base)
	}
	if result.headSHA, err = s.git.ResolveCommit(ctx, dir, input.Head); err != nil {
		return compareRange{}, notFound("unknown ref " + input.Head)
	}
	result.from = result.baseSHA
	if mergeBase, err := s.git.MergeBase(ctx, dir, result.baseSHA, result.headSHA); err == nil {
		result.mergeBase, result.from = mergeBase, mergeBase
	}
	return result, nil
}

func (s *Server) handleCompare(r *http.Request, input *compareInput) (*body[compareView], error) {
	comparison, err := s.resolveCompare(r, input)
	if err != nil {
		return nil, err
	}
	view := compareView{BaseSHA: comparison.baseSHA, HeadSHA: comparison.headSHA, MergeBase: comparison.mergeBase}
	if view.Commits, err = s.commits(r.Context(), comparison.repo, view.HeadSHA, 250, view.BaseSHA); err != nil {
		return nil, err
	}
	view.Files, err = s.git.DiffFiles(r.Context(), s.repoDir(comparison.repo), comparison.from, view.HeadSHA)
	if err != nil {
		return nil, err
	}
	return reply(view)
}

func (s *Server) handleCompareDiff(r *http.Request, input *compareInput) (*diffOutput, error) {
	comparison, err := s.resolveCompare(r, input)
	if err != nil {
		return nil, err
	}
	diff, err := s.git.Diff(r.Context(), s.repoDir(comparison.repo), comparison.from, comparison.headSHA)
	if err != nil {
		return nil, err
	}
	return &diffOutput{ContentType: "text/x-diff; charset=utf-8", Body: []byte(diff)}, nil
}
