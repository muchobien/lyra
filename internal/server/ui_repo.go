package server

import (
	"context"
	"html/template"
	"mime"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/muchobien/lyra/internal/repos"
)

const (
	maxReadmeSize    = 256 << 10
	maxHighlightSize = 1 << 20
)

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
		if !strings.HasSuffix(strings.ToLower(entry.Name), ".md") {
			continue
		}
		if rendered, err := renderMarkdownFile(content); err == nil {
			shown = template.HTML(rendered)
		}
	}
	return shown
}

var archiveTypes = map[string]string{".tar.gz": "application/gzip", ".zip": "application/zip"}

func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead, routeRepo(r))
	if err != nil {
		return err
	}
	target := pathParam(r, "*")
	for suffix, contentType := range archiveTypes {
		ref, found := strings.CutSuffix(target, suffix)
		if !found {
			continue
		}
		dir := s.repoDir(repo.Name)
		sha, err := s.git.ResolveCommit(r.Context(), dir, ref)
		if err != nil {
			return notFound("unknown ref " + ref)
		}
		short := strings.TrimPrefix(strings.TrimPrefix(ref, "refs/heads/"), "refs/tags/")
		name := path.Base(repo.Name) + "-" + strings.ReplaceAll(short, "/", "-")
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
			"filename": name + suffix,
		}))
		return s.git.Archive(r.Context(), dir, strings.TrimPrefix(suffix, "."), name+"/", sha, w)
	}
	return notFound("archives end in .tar.gz or .zip")
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
