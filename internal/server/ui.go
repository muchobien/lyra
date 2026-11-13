package server

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"

	"github.com/muchobien/lyra/internal/store"
)

//go:embed templates
var templateFiles embed.FS

type page struct {
	Title    string
	User     string
	Repo     *store.Repo
	Ref      string
	CloneURL string
	Data     any
}

var pageNames = []string{
	"repos",
	"tree",
	"blob",
	"commits",
	"commit",
	"tags",
	"items",
	"item",
	"releases",
	"images",
	"runs",
	"run",
	"error",
}

var sizeUnits = []string{"B", "KB", "MB", "GB", "TB"}

var refPrefixes = []string{"refs/heads/", "refs/tags/", "refs/pull/"}

var templateFuncs = template.FuncMap{
	"short":     shortSHA,
	"esc":       url.PathEscape,
	"base":      path.Base,
	"ownerOf":   ownerOf,
	"pathesc":   escapePath,
	"join":      joinPath,
	"digestHex": digestHex,
	"refName":   refName,
	"duration":  formatDuration,
	"size":      formatSize,
	"day":       formatDay,
	"markdown":  renderMarkdown,
	"codeCSS":   func() template.CSS { return codeCSS },
}

var pages = map[string]*template.Template{}

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

var codeCSS template.CSS

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func ownerOf(name string) string {
	owner, _, _ := strings.Cut(name, "/")
	return owner
}

func escapePath(path string) string {
	segments := strings.Split(path, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

func joinPath(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

func digestHex(digest string) string {
	_, hex, found := strings.Cut(digest, ":")
	if !found {
		return digest
	}
	return hex
}

func refName(ref string) string {
	for _, prefix := range refPrefixes {
		if name, found := strings.CutPrefix(ref, prefix); found {
			return name
		}
	}
	return ref
}

func formatDuration(start, stop *time.Time) string {
	end := time.Now()
	if stop != nil {
		end = *stop
	}
	elapsed := end.Sub(*start).Round(time.Second)
	if elapsed < time.Minute {
		return elapsed.String()
	}
	return fmt.Sprintf("%dm %ds", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
}

func formatSize(bytes int64) string {
	value, unit := float64(bytes), 0
	for value >= 1000 && unit < len(sizeUnits)-1 {
		value /= 1000
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d B", bytes)
	}
	return fmt.Sprintf("%.1f %s", value, sizeUnits[unit])
}

func formatDay(date string) string {
	if len(date) >= 16 {
		return strings.Replace(date[:16], "T", " ", 1)
	}
	return date
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

func init() {
	var light, dark strings.Builder
	codeFormatter.WriteCSS(&light, styles.Get("github"))
	codeFormatter.WriteCSS(&dark, styles.Get("github-dark"))
	codeCSS = template.CSS(light.String() + "\n@media (prefers-color-scheme: dark) {\n" + dark.String() + "}\n")
	for _, name := range pageNames {
		layout := template.New("layout.html").Funcs(templateFuncs)
		pages[name] = template.Must(layout.ParseFS(templateFiles, "templates/layout.html", "templates/"+name+".html"))
	}
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, name string, data page) {
	data.User = identityFrom(r).actor.User
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := pages[name].Execute(w, data); err != nil {
		slog.Error("render", "page", name, "err", err)
	}
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	s.render(w, r, status, "error", page{Title: http.StatusText(status), Data: message})
}

func (s *Server) repoPage(r *http.Request, repo *store.Repo, title, ref string, data any) page {
	return page{Title: title, Repo: repo, Ref: ref, CloneURL: cloneURL(r, repo.Name), Data: data}
}

func (s *Server) uiLookup(r *http.Request) (store.Repo, error) {
	name := r.PathValue("owner") + "/" + r.PathValue("repo")
	if !identityFrom(r).can(scopeRead, name) {
		return store.Repo{}, forbidden("This token cannot read " + name + ".")
	}
	repo, err := s.store.Repo(r.Context(), name)
	return repo, missing(err, "Repository "+name+" not found.")
}
