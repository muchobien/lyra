package server

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/webui"
)

type treeResponse struct {
	Ref     string        `json:"ref"`
	SHA     string        `json:"sha"`
	Path    string        `json:"path"`
	Empty   bool          `json:"empty"`
	Entries []repos.Entry `json:"entries"`
	Last    *repos.Commit `json:"last"`
	Readme  string        `json:"readme"`
}

type blobResponse struct {
	Path   string `json:"path"`
	Size   int    `json:"size"`
	Code   string `json:"code"`
	Binary bool   `json:"binary"`
}

func (s *Server) uiApplication(w http.ResponseWriter, r *http.Request) error {
	if r.PathValue("repo") != "" {
		if _, err := s.lookupRepo(r, scopeRead); err != nil {
			return err
		}
	} else if owner := r.PathValue("owner"); owner != "" {
		groups, err := s.ownerGroups(r.Context(), identityFrom(r), owner)
		if err != nil {
			return err
		}
		if len(groups) == 0 {
			return notFound("owner " + owner + " not found")
		}
	}
	webui.Serve(w, r)
	return nil
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	dir := s.repoDir(repo.Name)
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, dir)
	}
	path := strings.Trim(r.URL.Query().Get("path"), "/")
	view := treeResponse{Ref: ref, Path: path, Entries: []repos.Entry{}}
	view.SHA, err = s.git.ResolveCommit(ctx, dir, ref)
	if err != nil {
		branches, branchErr := s.git.Branches(ctx, dir)
		if branchErr != nil {
			return branchErr
		}
		if len(branches) != 0 {
			return notFound("unknown ref " + ref)
		}
		view.Empty = true
		return writeJSON(w, http.StatusOK, view)
	}
	view.Entries, err = s.git.Tree(ctx, dir, view.SHA, path)
	if errors.Is(err, repos.ErrNotFound) {
		return notFound("directory not found")
	}
	if err != nil {
		return err
	}
	view.Readme = string(s.readme(ctx, dir, view.SHA, path, view.Entries))
	commits, err := s.commits(ctx, repo.Name, view.SHA, 1)
	if err != nil {
		return err
	}
	if len(commits) != 0 {
		view.Last = &commits[0]
	}
	return writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleBlob(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	dir := s.repoDir(repo.Name)
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, dir)
	}
	sha, err := s.git.ResolveCommit(ctx, dir, ref)
	if err != nil {
		return notFound("unknown ref " + ref)
	}
	path := strings.Trim(r.URL.Query().Get("path"), "/")
	content, err := s.git.Blob(ctx, dir, sha, path)
	if errors.Is(err, repos.ErrNotFound) {
		return notFound("file not found")
	}
	if err != nil {
		return err
	}
	if r.URL.Query().Has("raw") {
		writeRaw(w, content)
		return nil
	}
	view := blobResponse{
		Path: path, Size: len(content), Binary: !utf8.Valid(content) || len(content) > maxHighlightSize,
	}
	if !view.Binary {
		lexer := lexers.Match(path)
		if lexer == nil {
			lexer = lexers.Analyse(string(content))
		}
		view.Code = string(highlight(codeFormatter, lexer, string(content)))
	}
	return writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	dir := s.repoDir(repo.Name)
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, dir)
	}
	sha, err := s.git.ResolveCommit(ctx, dir, ref)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return notFound("unknown ref " + ref)
		}
		return err
	}
	files, err := s.git.Files(ctx, dir, sha)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, files)
}

func (s *Server) handleCodeCSS(w http.ResponseWriter, r *http.Request) error {
	var light, dark strings.Builder
	if err := codeFormatter.WriteCSS(&light, styles.Get("github")); err != nil {
		return err
	}
	if err := codeFormatter.WriteCSS(&dark, styles.Get("github-dark")); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	css := ":root[data-theme=light] {\n" + light.String() +
		"}\n:root[data-theme=dark] {\n" + dark.String() +
		"}\n@media (prefers-color-scheme: light) {\n:root:not([data-theme=dark]) {\n" + light.String() +
		"}\n}\n@media (prefers-color-scheme: dark) {\n:root:not([data-theme=light]) {\n" + dark.String() + "}\n}\n"
	_, err := w.Write([]byte(css))
	return err
}
