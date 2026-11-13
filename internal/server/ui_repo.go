package server

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

const (
	commitsPerPage   = 50
	maxReadmeSize    = 256 << 10
	maxHighlightSize = 1 << 20
)

type crumb struct {
	Name string
	Path string
}

type treeView struct {
	Empty    bool
	Path     string
	Crumbs   []crumb
	Entries  []repos.Entry
	Branches []repos.Ref
	Last     *repos.Commit
	Readme   template.HTML
}

type blobView struct {
	Path   string
	Crumbs []crumb
	Size   int
	Code   template.HTML
	Binary bool
}

type commitsView struct {
	Commits []repos.Commit
	Older   string
	Checks  map[string]string
}

type commitView struct {
	Commit repos.Commit
	Patch  template.HTML
	Checks store.Checks
}

func crumbs(path string) []crumb {
	trail := []crumb{}
	if path == "" {
		return trail
	}
	parts := strings.Split(path, "/")
	for index, part := range parts {
		trail = append(trail, crumb{Name: part, Path: strings.Join(parts[:index+1], "/")})
	}
	return trail
}

func (s *Server) uiTree(w http.ResponseWriter, r *http.Request) error {
	return s.showTree(w, r, r.PathValue("ref"), strings.Trim(r.PathValue("*"), "/"))
}

func isReadme(entry repos.Entry) bool {
	return entry.Type == "blob" &&
		strings.EqualFold(strings.TrimSuffix(entry.Name, ".md"), "readme") &&
		entry.Size < maxReadmeSize
}

func (s *Server) readme(ctx context.Context, dir, sha, path string, entries []repos.Entry) template.HTML {
	var shown template.HTML
	for _, entry := range entries {
		if !isReadme(entry) {
			continue
		}
		content, err := s.git.Blob(ctx, dir, sha, strings.TrimPrefix(path+"/"+entry.Name, "/"))
		if err != nil || !utf8.Valid(content) {
			continue
		}
		shown = preformatted(string(content))
		var rendered bytes.Buffer
		if strings.HasSuffix(strings.ToLower(entry.Name), ".md") && markdown.Convert(content, &rendered) == nil {
			shown = template.HTML(`<article class="markdown">` + rendered.String() + `</article>`)
		}
	}
	return shown
}

func (s *Server) showTree(w http.ResponseWriter, r *http.Request, ref, path string) error {
	ctx := r.Context()
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	dir := s.repoDir(repo.Name)
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, dir)
	}
	branches, err := s.git.Branches(ctx, dir)
	if err != nil {
		return err
	}
	view := treeView{Path: path, Branches: branches, Crumbs: crumbs(path)}
	sha, err := s.git.ResolveCommit(ctx, dir, ref)
	if err != nil && len(branches) > 0 {
		return notFound("Unknown ref " + ref + ".")
	}
	if err != nil {
		view.Empty = true
		s.render(w, r, http.StatusOK, "tree", s.repoPage(r, &repo, repo.Name, ref, view))
		return nil
	}
	view.Entries, err = s.git.Tree(ctx, dir, sha, path)
	if errors.Is(err, repos.ErrNotFound) {
		return notFound("No directory " + path + " at " + ref + ".")
	}
	if err != nil {
		return err
	}
	view.Readme = s.readme(ctx, dir, sha, path, view.Entries)
	if commits, err := s.commits(ctx, repo.Name, sha, 1); err == nil && len(commits) == 1 {
		view.Last = &commits[0]
	}
	title := repo.Name
	if path != "" {
		title = path + " · " + repo.Name
	}
	s.render(w, r, http.StatusOK, "tree", s.repoPage(r, &repo, title, ref, view))
	return nil
}

func writeRaw(w http.ResponseWriter, content []byte) {
	w.Header().Set("Content-Type", "application/octet-stream")
	if utf8.Valid(content) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox")
	w.Write(content)
}

func (s *Server) uiBlob(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	dir := s.repoDir(repo.Name)
	ref := r.PathValue("ref")
	path := strings.Trim(r.PathValue("*"), "/")
	sha, err := s.git.ResolveCommit(ctx, dir, ref)
	if err != nil {
		return notFound("Unknown ref " + ref + ".")
	}
	content, err := s.git.Blob(ctx, dir, sha, path)
	if errors.Is(err, repos.ErrNotFound) {
		return notFound("No file " + path + " at " + ref + ".")
	}
	if err != nil {
		return err
	}
	if r.URL.Query().Has("raw") {
		writeRaw(w, content)
		return nil
	}
	view := blobView{Path: path, Crumbs: crumbs(path), Size: len(content)}
	if utf8.Valid(content) && len(content) <= maxHighlightSize {
		lexer := lexers.Match(path)
		if lexer == nil {
			lexer = lexers.Analyse(string(content))
		}
		view.Code = highlight(codeFormatter, lexer, string(content))
	} else {
		view.Binary = true
	}
	s.render(w, r, http.StatusOK, "blob", s.repoPage(r, &repo, path+" · "+repo.Name, ref, view))
	return nil
}

func (s *Server) checkStates(ctx context.Context, repo string, commits []repos.Commit) (map[string]string, error) {
	states := map[string]string{}
	for _, commit := range commits {
		checks, err := s.store.Checks(ctx, repo, commit.SHA)
		if err != nil {
			return nil, err
		}
		if checks.State != checksNone {
			states[commit.SHA] = checks.State
		}
	}
	return states, nil
}

func (s *Server) uiCommits(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	ref := r.PathValue("ref")
	commits, err := s.commits(ctx, repo.Name, ref, commitsPerPage)
	if errors.Is(err, repos.ErrNotFound) {
		return notFound("Unknown ref " + ref + ".")
	}
	if err != nil {
		return err
	}
	view := commitsView{Commits: commits}
	if view.Checks, err = s.checkStates(ctx, repo.Name, commits); err != nil {
		return err
	}
	if len(commits) == commitsPerPage && len(commits[len(commits)-1].Parents) > 0 {
		view.Older = commits[len(commits)-1].Parents[0]
	}
	s.render(w, r, http.StatusOK, "commits", s.repoPage(r, &repo, "Commits · "+repo.Name, ref, view))
	return nil
}

func (s *Server) uiCommit(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	dir := s.repoDir(repo.Name)
	commit, err := s.git.Commit(ctx, dir, r.PathValue("sha"))
	if errors.Is(err, repos.ErrNotFound) {
		return notFound("Unknown commit.")
	}
	if err != nil {
		return err
	}
	if err := s.verifySignatures(ctx, &commit.Signature); err != nil {
		return err
	}
	patch, err := s.git.Patch(ctx, dir, commit.SHA)
	if err != nil {
		return err
	}
	checks, err := s.store.Checks(ctx, repo.Name, commit.SHA)
	if err != nil {
		return err
	}
	view := commitView{Commit: commit, Patch: highlight(patchFormatter, lexers.Get("diff"), patch), Checks: checks}
	s.render(w, r, http.StatusOK, "commit", s.repoPage(r, &repo, commit.Subject+" · "+repo.Name, commit.SHA, view))
	return nil
}

func (s *Server) uiTags(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	tags, err := s.tags(ctx, repo.Name)
	if err != nil {
		return err
	}
	ref := s.git.DefaultBranch(ctx, s.repoDir(repo.Name))
	s.render(w, r, http.StatusOK, "tags", s.repoPage(r, &repo, "Tags · "+repo.Name, ref, tags))
	return nil
}
