package server

import (
	"bytes"
	"context"
	"html/template"
	"net/http"
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
		var rendered bytes.Buffer
		if strings.HasSuffix(strings.ToLower(entry.Name), ".md") && markdown.Convert(content, &rendered) == nil {
			shown = template.HTML(`<article class="markdown">` + rendered.String() + `</article>`)
		}
	}
	return shown
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
