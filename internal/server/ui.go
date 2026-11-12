package server

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

//go:embed templates
var templateFiles embed.FS

const commitsPerPage = 50

type crumb struct {
	Name string
	Path string
}

type page struct {
	Title    string
	User     string
	Repo     *store.Repo
	Ref      string
	CloneURL string
	Data     any
}

var templateFuncs = template.FuncMap{
	"short": func(sha string) string {
		if len(sha) > 7 {
			return sha[:7]
		}
		return sha
	},
	"esc": url.PathEscape,
	"pathesc": func(path string) string {
		segments := strings.Split(path, "/")
		for index, segment := range segments {
			segments[index] = url.PathEscape(segment)
		}
		return strings.Join(segments, "/")
	},
	"join": func(dir, name string) string {
		if dir == "" {
			return name
		}
		return dir + "/" + name
	},
	"day": func(date string) string {
		if len(date) >= 16 {
			return strings.Replace(date[:16], "T", " ", 1)
		}
		return date
	},
}

var pages = map[string]*template.Template{}

var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

var codeFormatter = chromahtml.New(chromahtml.WithClasses(true), chromahtml.WithLineNumbers(true), chromahtml.LineNumbersInTable(true))

var patchFormatter = chromahtml.New(chromahtml.WithClasses(true))

var codeCSS template.CSS

func highlight(formatter *chromahtml.Formatter, lexer chroma.Lexer, source string) template.HTML {
	if lexer == nil {
		lexer = lexers.Fallback
	}
	iterator, err := chroma.Coalesce(lexer).Tokenise(nil, source)
	if err != nil {
		return template.HTML("<pre>" + template.HTMLEscapeString(source) + "</pre>")
	}
	var output strings.Builder
	if err := formatter.Format(&output, styles.Get("github"), iterator); err != nil {
		return template.HTML("<pre>" + template.HTMLEscapeString(source) + "</pre>")
	}
	return template.HTML(output.String())
}

func init() {
	var light, dark strings.Builder
	codeFormatter.WriteCSS(&light, styles.Get("github"))
	codeFormatter.WriteCSS(&dark, styles.Get("github-dark"))
	codeCSS = template.CSS(light.String() + "\n@media (prefers-color-scheme: dark) {\n" + dark.String() + "}\n")
	templateFuncs["codeCSS"] = func() template.CSS {
		return codeCSS
	}

	for _, name := range []string{"repos", "tree", "blob", "commits", "commit", "tags", "error"} {
		pages[name] = template.Must(template.New("layout.html").Funcs(templateFuncs).ParseFS(templateFiles, "templates/layout.html", "templates/"+name+".html"))
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

func (s *Server) uiLookup(w http.ResponseWriter, r *http.Request) (store.Repo, bool) {
	name := r.PathValue("repo")
	if !identityFrom(r).can("read", name) {
		s.renderError(w, r, http.StatusForbidden, "This token cannot read "+name+".")
		return store.Repo{}, false
	}
	repo, err := s.store.Repo(name)
	if errors.Is(err, store.ErrNotFound) {
		s.renderError(w, r, http.StatusNotFound, "Repository "+name+" not found.")
		return repo, false
	}
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return repo, false
	}
	return repo, true
}

func (s *Server) uiRepoList(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	all, err := s.store.Repos()
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	visible := []store.Repo{}
	for _, repo := range all {
		if id.can("read", repo.Name) {
			visible = append(visible, repo)
		}
	}
	s.render(w, r, http.StatusOK, "repos", page{Title: "Repositories", Data: visible})
}

func (s *Server) uiRepo(w http.ResponseWriter, r *http.Request) {
	s.showTree(w, r, "", "")
}

func (s *Server) uiTree(w http.ResponseWriter, r *http.Request) {
	s.showTree(w, r, r.PathValue("ref"), strings.Trim(r.PathValue("path"), "/"))
}

func (s *Server) showTree(w http.ResponseWriter, r *http.Request, ref, path string) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	dir := s.repoDir(repo.Name)
	if ref == "" {
		ref = s.git.DefaultBranch(dir)
	}
	branches, err := s.git.Branches(dir)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	view := struct {
		Empty    bool
		Path     string
		Crumbs   []crumb
		Entries  []repos.Entry
		Branches []repos.Ref
		Last     *repos.Commit
		Readme   template.HTML
	}{Path: path, Branches: branches}
	base := page{Title: repo.Name, Repo: &repo, Ref: ref, CloneURL: cloneURL(r, repo.Name)}
	sha, err := s.git.ResolveCommit(dir, ref)
	if err != nil {
		if len(branches) == 0 {
			view.Empty = true
			base.Data = view
			s.render(w, r, http.StatusOK, "tree", base)
			return
		}
		s.renderError(w, r, http.StatusNotFound, "Unknown ref "+ref+".")
		return
	}
	entries, err := s.git.Tree(dir, sha, path)
	if errors.Is(err, repos.ErrNotFound) {
		s.renderError(w, r, http.StatusNotFound, "No directory "+path+" at "+ref+".")
		return
	}
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	view.Entries = entries
	if path != "" {
		parts := strings.Split(path, "/")
		for index, part := range parts {
			view.Crumbs = append(view.Crumbs, crumb{Name: part, Path: strings.Join(parts[:index+1], "/")})
		}
	}
	for _, entry := range entries {
		if entry.Type == "blob" && strings.EqualFold(strings.TrimSuffix(entry.Name, ".md"), "readme") && entry.Size < 256<<10 {
			content, err := s.git.Blob(dir, sha, strings.TrimPrefix(path+"/"+entry.Name, "/"))
			if err != nil || !utf8.Valid(content) {
				continue
			}
			view.Readme = template.HTML("<pre>" + template.HTMLEscapeString(string(content)) + "</pre>")
			var rendered bytes.Buffer
			if strings.HasSuffix(strings.ToLower(entry.Name), ".md") && markdown.Convert(content, &rendered) == nil {
				view.Readme = template.HTML(`<article class="markdown">` + rendered.String() + `</article>`)
			}
		}
	}
	commits, err := s.commits(repo.Name, sha, 1)
	if err == nil && len(commits) == 1 {
		view.Last = &commits[0]
	}
	base.Data = view
	if path != "" {
		base.Title = path + " · " + repo.Name
	}
	s.render(w, r, http.StatusOK, "tree", base)
}

func (s *Server) uiBlob(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	dir := s.repoDir(repo.Name)
	ref := r.PathValue("ref")
	path := strings.Trim(r.PathValue("path"), "/")
	sha, err := s.git.ResolveCommit(dir, ref)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "Unknown ref "+ref+".")
		return
	}
	content, err := s.git.Blob(dir, sha, path)
	if errors.Is(err, repos.ErrNotFound) {
		s.renderError(w, r, http.StatusNotFound, "No file "+path+" at "+ref+".")
		return
	}
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if r.URL.Query().Has("raw") {
		w.Header().Set("Content-Type", "application/octet-stream")
		if utf8.Valid(content) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "sandbox")
		w.Write(content)
		return
	}
	parts := strings.Split(path, "/")
	crumbs := []crumb{}
	for index, part := range parts {
		crumbs = append(crumbs, crumb{Name: part, Path: strings.Join(parts[:index+1], "/")})
	}
	view := struct {
		Path   string
		Crumbs []crumb
		Size   int
		Code   template.HTML
		Binary bool
	}{Path: path, Crumbs: crumbs, Size: len(content)}
	if utf8.Valid(content) && len(content) <= 1<<20 {
		lexer := lexers.Match(path)
		if lexer == nil {
			lexer = lexers.Analyse(string(content))
		}
		view.Code = highlight(codeFormatter, lexer, string(content))
	} else {
		view.Binary = true
	}
	s.render(w, r, http.StatusOK, "blob", page{Title: path + " · " + repo.Name, Repo: &repo, Ref: ref, CloneURL: cloneURL(r, repo.Name), Data: view})
}

func (s *Server) uiCommits(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	ref := r.PathValue("ref")
	commits, err := s.commits(repo.Name, ref, commitsPerPage)
	if errors.Is(err, repos.ErrNotFound) {
		s.renderError(w, r, http.StatusNotFound, "Unknown ref "+ref+".")
		return
	}
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	view := struct {
		Commits []repos.Commit
		Older   string
	}{Commits: commits}
	if len(commits) == commitsPerPage && len(commits[len(commits)-1].Parents) > 0 {
		view.Older = commits[len(commits)-1].Parents[0]
	}
	s.render(w, r, http.StatusOK, "commits", page{Title: "Commits · " + repo.Name, Repo: &repo, Ref: ref, CloneURL: cloneURL(r, repo.Name), Data: view})
}

func (s *Server) uiCommit(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	dir := s.repoDir(repo.Name)
	commit, err := s.git.Commit(dir, r.PathValue("sha"))
	if errors.Is(err, repos.ErrNotFound) {
		s.renderError(w, r, http.StatusNotFound, "Unknown commit.")
		return
	}
	if err == nil {
		err = s.attachKeys(&commit.Signature)
	}
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	patch, err := s.git.Patch(dir, commit.SHA)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	view := struct {
		Commit repos.Commit
		Patch  template.HTML
	}{Commit: commit, Patch: highlight(patchFormatter, lexers.Get("diff"), patch)}
	s.render(w, r, http.StatusOK, "commit", page{Title: commit.Subject + " · " + repo.Name, Repo: &repo, Ref: commit.SHA, CloneURL: cloneURL(r, repo.Name), Data: view})
}

func (s *Server) uiTags(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	tags, err := s.tags(repo.Name)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	s.render(w, r, http.StatusOK, "tags", page{Title: "Tags · " + repo.Name, Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: tags})
}
