package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

var itemPaths = map[string]string{"issue": "issues", "pr": "pulls"}

type itemRow struct {
	Item store.Item
	Pull *store.Pull
}

type itemPage struct {
	Kind      string
	Path      string
	Noun      string
	Item      store.Item
	Pull      *store.Pull
	Comments  []store.Comment
	CanWrite  bool
	Labels    string
	Assignees string
	Tab       string
	Commits   []repos.Commit
	Files     []repos.FileChange
	Diff      template.HTML
}

func splitList(text string) []string {
	values := []string{}
	for _, value := range strings.Split(text, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func difference(values, without []string) []string {
	remaining := []string{}
	for _, value := range values {
		if !slices.Contains(without, value) {
			remaining = append(remaining, value)
		}
	}
	return remaining
}

func (s *Server) renderStoreError(w http.ResponseWriter, r *http.Request, err error, missing string) {
	var invalid store.InvalidError
	switch {
	case errors.Is(err, store.ErrNotFound):
		s.renderError(w, r, http.StatusNotFound, missing)
	case errors.As(err, &invalid):
		s.renderError(w, r, http.StatusBadRequest, invalid.Error())
	default:
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
	}
}

func (s *Server) uiItem(w http.ResponseWriter, r *http.Request, kind string, write bool) (store.Repo, store.Item, bool) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return repo, store.Item{}, false
	}
	if write && !identityFrom(r).can("write", repo.Name) {
		s.renderError(w, r, http.StatusForbidden, "This token cannot write to "+repo.Name+".")
		return repo, store.Item{}, false
	}
	missing := strings.ToUpper(nouns[kind][:1]) + nouns[kind][1:] + " not found."
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, missing)
		return repo, store.Item{}, false
	}
	item, err := s.store.Item(kind, repo.Name, number)
	if err != nil {
		s.renderStoreError(w, r, err, missing)
		return repo, item, false
	}
	return repo, item, true
}

func (s *Server) uiItems(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := s.uiLookup(w, r)
		if !ok {
			return
		}
		state := r.URL.Query().Get("state")
		if state != "closed" {
			state = "open"
		}
		rows := []itemRow{}
		if kind == "issue" {
			items, err := s.store.Items(kind, repo.Name, state)
			if err != nil {
				s.renderError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			for _, item := range items {
				rows = append(rows, itemRow{Item: item})
			}
		} else {
			pulls, err := s.store.Pulls(repo.Name, "all")
			if err != nil {
				s.renderError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			for _, pull := range pulls {
				if (pull.State == "open") == (state == "open") {
					rows = append(rows, itemRow{Item: pull.Item, Pull: &pull})
				}
			}
		}
		view := struct {
			Kind     string
			Path     string
			Noun     string
			Rows     []itemRow
			State    string
			CanWrite bool
		}{kind, itemPaths[kind], nouns[kind], rows, state, identityFrom(r).can("write", repo.Name)}
		title := "Issues · "
		if kind == "pr" {
			title = "Pull requests · "
		}
		s.render(w, r, http.StatusOK, "items", page{Title: title + repo.Name, Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: view})
	}
}

func (s *Server) uiCreateIssue(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	id := identityFrom(r)
	if !id.can("write", repo.Name) {
		s.renderError(w, r, http.StatusForbidden, "This token cannot write to "+repo.Name+".")
		return
	}
	title, body := r.FormValue("title"), r.FormValue("body")
	if problem := checkText(&title, &body); problem != "" {
		s.renderError(w, r, http.StatusBadRequest, problem)
		return
	}
	item, _, err := s.store.CreateItem("issue", repo.Name, id.actor, title, body, nil, nil)
	if err != nil {
		s.renderStoreError(w, r, err, "Repository not found.")
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/issues/%d", repo.Name, item.Number), http.StatusSeeOther)
}

func (s *Server) uiItemPage(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, item, ok := s.uiItem(w, r, kind, false)
		if !ok {
			return
		}
		view := itemPage{
			Kind: kind, Path: itemPaths[kind], Noun: nouns[kind], Item: item, CanWrite: identityFrom(r).can("write", repo.Name),
			Labels: strings.Join(item.Labels, ", "), Assignees: strings.Join(item.Assignees, ", "), Tab: r.URL.Query().Get("tab"),
		}
		var err error
		if kind == "pr" {
			pull, pullErr := s.store.Pull(repo.Name, item.Number)
			if pullErr != nil {
				s.renderError(w, r, http.StatusInternalServerError, pullErr.Error())
				return
			}
			view.Pull = &pull
			base, head := s.pullRange(repo.Name, pull)
			switch view.Tab {
			case "commits":
				view.Commits, err = s.commits(repo.Name, head, 250, base)
			case "files":
				dir := s.repoDir(repo.Name)
				if view.Files, err = s.git.DiffFiles(dir, base, head); err == nil {
					var diff string
					diff, err = s.git.Diff(dir, base, head)
					view.Diff = highlight(patchFormatter, lexers.Get("diff"), diff)
				}
			}
		}
		if view.Tab != "commits" && view.Tab != "files" {
			view.Tab = ""
			view.Comments, err = s.store.Comments(item)
		}
		if err != nil {
			s.renderError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		s.render(w, r, http.StatusOK, "item", page{Title: fmt.Sprintf("%s · #%d · %s", item.Title, item.Number, repo.Name), Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: view})
	}
}

func (s *Server) uiCommentItem(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, item, ok := s.uiItem(w, r, kind, true)
		if !ok {
			return
		}
		body := r.FormValue("body")
		if strings.TrimSpace(body) == "" {
			s.renderError(w, r, http.StatusBadRequest, "Comment body is empty.")
			return
		}
		if problem := checkText(nil, &body); problem != "" {
			s.renderError(w, r, http.StatusBadRequest, problem)
			return
		}
		if _, _, err := s.store.AddComment(kind, repo.Name, item.Number, identityFrom(r).actor, body); err != nil {
			s.renderStoreError(w, r, err, "Not found.")
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/%s/%s/%d", repo.Name, itemPaths[kind], item.Number), http.StatusSeeOther)
	}
}

func (s *Server) uiEditItem(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, item, ok := s.uiItem(w, r, kind, true)
		if !ok {
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		id := identityFrom(r)
		edit := store.ItemEdit{}
		if r.PostForm.Has("state") {
			state := r.PostForm.Get("state")
			edit.State = &state
		}
		if r.PostForm.Has("labels") {
			wanted := splitList(r.PostForm.Get("labels"))
			edit.AddLabels, edit.RemoveLabels = difference(wanted, item.Labels), difference(item.Labels, wanted)
		}
		if r.PostForm.Has("assignees") {
			wanted := id.resolve(splitList(r.PostForm.Get("assignees")))
			edit.AddAssignees, edit.RemoveAssignees = difference(wanted, item.Assignees), difference(item.Assignees, wanted)
		}
		if _, _, err := s.store.EditItem(kind, repo.Name, item.Number, id.actor, edit); err != nil {
			s.renderStoreError(w, r, err, "Not found.")
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/%s/%s/%d", repo.Name, itemPaths[kind], item.Number), http.StatusSeeOther)
	}
}
