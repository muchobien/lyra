package server

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"

	"github.com/muchobien/lyra/internal/store"
	"github.com/muchobien/lyra/internal/webui"
)

var markdown = goldmark.New(goldmark.WithExtensions(
	extension.GFM,
	highlighting.NewHighlighting(highlighting.WithFormatOptions(chromahtml.WithClasses(true))),
))

var codeFormatter = chromahtml.New(
	chromahtml.WithClasses(true),
	chromahtml.WithLineNumbers(true),
	chromahtml.LineNumbersInTable(true),
)

var patchFormatter = chromahtml.New(chromahtml.WithClasses(true))

func escapePath(path string) string {
	segments := strings.Split(path, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

func preformatted(source string) template.HTML {
	return template.HTML("<pre>" + template.HTMLEscapeString(source) + "</pre>")
}

func renderMarkdown(source string) template.HTML {
	var rendered bytes.Buffer
	if markdown.Convert([]byte(source), &rendered) != nil {
		return preformatted(source)
	}
	return template.HTML(rendered.String())
}

func highlight(formatter *chromahtml.Formatter, lexer chroma.Lexer, source string) template.HTML {
	if lexer == nil {
		lexer = lexers.Fallback
	}
	iterator, err := chroma.Coalesce(lexer).Tokenise(nil, source)
	if err != nil {
		return preformatted(source)
	}
	var output strings.Builder
	if err := formatter.Format(&output, styles.Get("github"), iterator); err != nil {
		return preformatted(source)
	}
	return template.HTML(output.String())
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	webui.Serve(w, r, status, message)
}

func (s *Server) uiLookup(r *http.Request) (store.Repo, error) {
	name := r.PathValue("owner") + "/" + r.PathValue("repo")
	if !identityFrom(r).can(scopeRead, name) {
		return store.Repo{}, forbidden("This token cannot read " + name + ".")
	}
	repo, err := s.store.Repo(r.Context(), name)
	return repo, missing(err, "Repository "+name+" not found.")
}
