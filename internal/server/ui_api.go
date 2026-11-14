package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/danielgtaylor/huma/v2"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/webui"
)

type sourceInput struct {
	RepoPath
	Ref  string `query:"ref" doc:"Branch, tag, or commit; defaults to the repository's default branch."`
	Path string `query:"path" doc:"Path relative to the repository root."`
}

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

func (treeResponse) TransformSchema(_ huma.Registry, schema *huma.Schema) *huma.Schema {
	schema.Properties["last"] = &huma.Schema{AnyOf: []*huma.Schema{
		schema.Properties["last"], {Type: "null"},
	}}
	return schema
}

func (s *Server) uiApplication(w http.ResponseWriter, r *http.Request) error {
	if r.PathValue("repo") != "" {
		repo, err := s.lookupRepo(r, scopeRead)
		if err != nil {
			return err
		}
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 4)
		if len(parts) > 2 {
			switch parts[2] {
			case "pulls", "issues":
				if r.PathValue("number") != "" {
					kind := kindIssue
					if parts[2] == "pulls" {
						kind = kindPull
					}
					if _, _, err := s.uiItem(r, kind); err != nil {
						return err
					}
				}
			case "runs":
				if r.PathValue("number") != "" {
					number, err := uiRunNumber(r)
					if err != nil {
						return err
					}
					if _, err := s.lookupRun(r, scopeRead, number); err != nil {
						return err
					}
				}
			case "commits":
				if _, err := s.git.ResolveCommit(r.Context(), s.repoDir(repo.Name), r.PathValue("ref")); err != nil {
					return notFound("unknown ref " + r.PathValue("ref"))
				}
			case "commit":
				if _, err := s.git.ResolveCommit(r.Context(), s.repoDir(repo.Name), r.PathValue("sha")); err != nil {
					return notFound("commit not found")
				}
			case "releases":
				if tag := pathParam(r, "*"); tag != "" {
					if _, err := s.releaseByTag(r, repo.Name, tag); err != nil {
						return missing(err, "release not found")
					}
				}
			}
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
	webui.Serve(w, r, http.StatusOK, "")
	return nil
}

func (s *Server) uiSource(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	ref := r.PathValue("ref")
	path := strings.Trim(r.PathValue("*"), "/")
	sha, err := s.git.ResolveCommit(r.Context(), s.repoDir(repo.Name), ref)
	if err != nil {
		return notFound("unknown ref " + ref)
	}
	query := url.Values{"ref": {ref}}
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 4)
	if len(parts) > 2 && parts[2] == "blob" {
		content, err := s.git.Blob(r.Context(), s.repoDir(repo.Name), sha, path)
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
		query.Set("file", path)
	} else {
		if _, err := s.git.Tree(r.Context(), s.repoDir(repo.Name), sha, path); err != nil {
			if errors.Is(err, repos.ErrNotFound) {
				return notFound("directory not found")
			}
			return err
		}
		query.Set("path", path)
	}
	http.Redirect(w, r, "/"+escapePath(repo.Name)+"?"+query.Encode(), http.StatusTemporaryRedirect)
	return nil
}

func (s *Server) handleTree(r *http.Request, input *sourceInput) (*body[treeResponse], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	dir := s.repoDir(repo.Name)
	ref := input.Ref
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, dir)
	}
	path := strings.Trim(input.Path, "/")
	view := treeResponse{Ref: ref, Path: path, Entries: []repos.Entry{}}
	view.SHA, err = s.git.ResolveCommit(ctx, dir, ref)
	if err != nil {
		branches, branchErr := s.git.Branches(ctx, dir)
		if branchErr != nil {
			return nil, branchErr
		}
		if len(branches) != 0 {
			return nil, notFound("unknown ref " + ref)
		}
		view.Empty = true
		return reply(view)
	}
	view.Entries, err = s.git.Tree(ctx, dir, view.SHA, path)
	if errors.Is(err, repos.ErrNotFound) {
		return nil, notFound("directory not found")
	}
	if err != nil {
		return nil, err
	}
	view.Readme = string(s.readme(ctx, dir, view.SHA, path, view.Entries))
	if commits, err := s.commits(ctx, repo.Name, view.SHA, 1); err == nil && len(commits) != 0 {
		view.Last = &commits[0]
	}
	return reply(view)
}

func (s *Server) handleBlob(r *http.Request, input *sourceInput) (*body[blobResponse], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	dir := s.repoDir(repo.Name)
	ref := input.Ref
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, dir)
	}
	sha, err := s.git.ResolveCommit(ctx, dir, ref)
	if err != nil {
		return nil, notFound("unknown ref " + ref)
	}
	path := strings.Trim(input.Path, "/")
	content, err := s.git.Blob(ctx, dir, sha, path)
	if errors.Is(err, repos.ErrNotFound) {
		return nil, notFound("file not found")
	}
	if err != nil {
		return nil, err
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
	return reply(view)
}

func (s *Server) handleFiles(r *http.Request, input *sourceInput) (*body[[]string], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	dir := s.repoDir(repo.Name)
	ref := input.Ref
	if ref == "" {
		ref = s.git.DefaultBranch(ctx, dir)
	}
	sha, err := s.git.ResolveCommit(ctx, dir, ref)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return nil, notFound("unknown ref " + ref)
		}
		return nil, err
	}
	files, err := s.git.Files(ctx, dir, sha)
	if err != nil {
		return nil, err
	}
	return reply(files)
}

var codeCSS, codeCSSTag string

func init() {
	var light, dark strings.Builder
	if err := codeFormatter.WriteCSS(&light, styles.Get("github")); err != nil {
		panic(err)
	}
	if err := codeFormatter.WriteCSS(&dark, styles.Get("github-dark")); err != nil {
		panic(err)
	}
	codeCSS = ":root[data-theme=light] {\n" + light.String() +
		"}\n:root[data-theme=dark] {\n" + dark.String() +
		"}\n@media (prefers-color-scheme: light) {\n:root:not([data-theme=dark]) {\n" + light.String() +
		"}\n}\n@media (prefers-color-scheme: dark) {\n:root:not([data-theme=light]) {\n" + dark.String() + "}\n}\n"
	sum := sha256.Sum256([]byte(codeCSS))
	codeCSSTag = `"` + hex.EncodeToString(sum[:8]) + `"`
}

func (s *Server) handleCodeCSS(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", codeCSSTag)
	http.ServeContent(w, r, "code.css", time.Time{}, strings.NewReader(codeCSS))
	return nil
}

func (s *Server) handleRawBlob(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	dir := s.repoDir(repo.Name)
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = s.git.DefaultBranch(r.Context(), dir)
	}
	sha, err := s.git.ResolveCommit(r.Context(), dir, ref)
	if err != nil {
		return notFound("unknown ref " + ref)
	}
	content, err := s.git.Blob(r.Context(), dir, sha, strings.Trim(r.URL.Query().Get("path"), "/"))
	if errors.Is(err, repos.ErrNotFound) {
		return notFound("file not found")
	}
	if err != nil {
		return err
	}
	writeRaw(w, content)
	return nil
}
