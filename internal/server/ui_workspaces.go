package server

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"code.forgejo.org/forgejo/runner/v13/act/model"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

type renderedComment struct {
	store.Comment
	HTML string `json:"html"`
}

type renderedReview struct {
	store.Review
	HTML string `json:"html"`
}

type itemWorkspace struct {
	Item       store.Item        `json:"item"`
	Pull       *store.Pull       `json:"pull"`
	HTML       string            `json:"html"`
	Comments   []renderedComment `json:"comments"`
	Reviews    []renderedReview  `json:"reviews"`
	Merge      *mergePlan        `json:"merge"`
	MergeError string            `json:"merge_error"`
	Checks     *store.Checks     `json:"checks"`
	CanWrite   bool              `json:"can_write"`
	IsAuthor   bool              `json:"is_author"`
}

type commitWorkspace struct {
	Commit repos.Commit `json:"commit"`
	Checks store.Checks `json:"checks"`
	Files  []diffFile   `json:"files"`
}

type workflowWorkspace struct {
	Path     string                   `json:"path"`
	Name     string                   `json:"name"`
	Dispatch bool                     `json:"dispatch"`
	Inputs   map[string]workflowInput `json:"inputs"`
	Error    string                   `json:"error,omitempty"`
}

type workflowInput struct {
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Default     string   `json:"default"`
	Type        string   `json:"type"`
	Options     []string `json:"options"`
}

type markdownRequest struct {
	Body string `json:"body"`
}

type renderedMarkdown struct {
	HTML string `json:"html"`
}

func (s *Server) handleMarkdown(_ *http.Request, input *bodyInput[markdownRequest]) (*body[renderedMarkdown], error) {
	body := input.Body
	if err := checkText(nil, &body.Body); err != nil {
		return nil, err
	}
	return reply(renderedMarkdown{HTML: string(renderMarkdown(body.Body))})
}

func (s *Server) handleItemWorkspace(kind string) func(*http.Request, *ItemPath) (*body[itemWorkspace], error) {
	return func(r *http.Request, _ *ItemPath) (*body[itemWorkspace], error) {
		repo, item, err := s.uiItem(r, kind, false)
		if err != nil {
			return nil, err
		}
		comments, err := s.store.Comments(r.Context(), item)
		if err != nil {
			return nil, err
		}
		reviews := []store.Review{}
		workspace := itemWorkspace{
			Item: item, HTML: string(renderMarkdown(item.Body)),
			Comments: []renderedComment{}, Reviews: []renderedReview{},
			CanWrite: identityFrom(r).can(scopeWrite, repo.Name),
		}
		if kind == kindPull {
			pull, err := s.store.Pull(r.Context(), repo.Name, item.Number)
			if err != nil {
				return nil, err
			}
			workspace.Pull = &pull
			workspace.IsAuthor = isAuthor(pull, identityFrom(r).actor)
			workspace.Merge, workspace.MergeError = s.mergeStatus(r.Context(), repo.Name, pull)
			checks, err := s.store.Checks(r.Context(), repo.Name, pull.HeadSHA)
			if err != nil {
				return nil, err
			}
			workspace.Checks = &checks
			if reviews, err = s.store.Reviews(r.Context(), item); err != nil {
				return nil, err
			}
			s.markOutdated(r.Context(), repo.Name, pull, comments)
			for index := range reviews {
				s.markOutdated(r.Context(), repo.Name, pull, reviews[index].Comments)
			}
		}
		for _, comment := range comments {
			workspace.Comments = append(workspace.Comments, renderedComment{
				Comment: comment, HTML: string(renderMarkdown(comment.Body)),
			})
		}
		for _, review := range reviews {
			workspace.Reviews = append(workspace.Reviews, renderedReview{
				Review: review, HTML: string(renderMarkdown(review.Body)),
			})
		}
		return reply(workspace)
	}
}

func highlightedDiff(files []*gitdiff.File, comments []store.Comment) []diffFile {
	view := diffView(files, comments)
	for fileIndex := range view {
		file := &view[fileIndex]
		lexer := lexers.Match(file.Path)
		for lineIndex := range file.Lines {
			line := &file.Lines[lineIndex]
			if line.Kind != "hunk" {
				line.HTML = string(highlight(patchFormatter, lexer, line.Text))
			}
		}
	}
	return view
}

func (s *Server) handlePullPatch(r *http.Request, input *ItemPath) (*body[[]diffFile], error) {
	repo, pull, err := s.lookupPull(r, input.Number)
	if err != nil {
		return nil, err
	}
	comments, err := s.store.Comments(r.Context(), pull.Item)
	if err != nil {
		return nil, err
	}
	s.markOutdated(r.Context(), repo.Name, pull, comments)
	files, err := s.pullDiff(r.Context(), repo.Name, pull)
	if err != nil {
		return nil, err
	}
	return reply(highlightedDiff(files, comments))
}

func (s *Server) handleCommitWorkspace(r *http.Request, input *commitInput) (*body[commitWorkspace], error) {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	dir := s.repoDir(repo.Name)
	commit, err := s.git.Commit(ctx, dir, input.Rev)
	if errors.Is(err, repos.ErrNotFound) {
		return nil, notFound("commit not found")
	}
	if err != nil {
		return nil, err
	}
	if err := s.verifySignatures(ctx, &commit.Signature); err != nil {
		return nil, err
	}
	patch, err := s.git.Patch(ctx, dir, commit.SHA)
	if err != nil {
		return nil, err
	}
	files, _, err := gitdiff.Parse(strings.NewReader(patch))
	if err != nil {
		return nil, err
	}
	checks, err := s.store.Checks(ctx, repo.Name, commit.SHA)
	if err != nil {
		return nil, err
	}
	return reply(commitWorkspace{Commit: commit, Checks: checks, Files: highlightedDiff(files, nil)})
}

func (s *Server) handleWorkflows(r *http.Request, input *sourceInput) (*body[[]workflowWorkspace], error) {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	dir := s.repoDir(repo.Name)
	ref := input.Ref
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, dir)
	}
	sha, err := s.git.ResolveCommit(ctx, dir, ref)
	if err != nil {
		branches, branchErr := s.git.Branches(ctx, dir)
		if branchErr != nil {
			return nil, branchErr
		}
		if len(branches) == 0 {
			return reply([]workflowWorkspace{})
		}
		return nil, notFound("unknown ref " + ref)
	}
	workflows := []workflowWorkspace{}
	for _, file := range workflowFiles(ctx, s.git, dir, sha) {
		view := workflowWorkspace{Path: file, Name: file, Inputs: map[string]workflowInput{}}
		content, err := s.git.Blob(ctx, dir, sha, file)
		if err != nil {
			return nil, err
		}
		workflow, err := model.ReadWorkflow(bytes.NewReader(content), false)
		if err != nil {
			view.Error = err.Error()
		} else {
			if workflow.Name != "" {
				view.Name = workflow.Name
			}
			if dispatch := workflow.WorkflowDispatchConfig(); dispatch != nil {
				view.Dispatch = true
				for name, input := range dispatch.Inputs {
					view.Inputs[name] = workflowInput{
						Description: input.Description, Required: input.Required, Default: input.Default,
						Type: input.Type, Options: input.Options,
					}
				}
			}
		}
		workflows = append(workflows, view)
	}
	return reply(workflows)
}
