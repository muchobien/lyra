package server

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/muchobien/lyra/internal/store"
)

type issueView struct {
	Item      store.Item
	Comments  []store.Comment
	CanWrite  bool
	Labels    string
	Assignees string
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

func (s *Server) uiItem(w http.ResponseWriter, r *http.Request, write bool) (store.Repo, store.Item, bool) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return repo, store.Item{}, false
	}
	if write && !identityFrom(r).can("write", repo.Name) {
		s.renderError(w, r, http.StatusForbidden, "This token cannot write to "+repo.Name+".")
		return repo, store.Item{}, false
	}
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "Issue not found.")
		return repo, store.Item{}, false
	}
	item, err := s.store.Item("issue", repo.Name, number)
	if err != nil {
		s.renderStoreError(w, r, err, "Issue not found.")
		return repo, item, false
	}
	return repo, item, true
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

func (s *Server) uiIssues(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	state := r.URL.Query().Get("state")
	if state != "closed" {
		state = "open"
	}
	items, err := s.store.Items("issue", repo.Name, state)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	view := struct {
		Items    []store.Item
		State    string
		CanWrite bool
	}{items, state, identityFrom(r).can("write", repo.Name)}
	s.render(w, r, http.StatusOK, "issues", page{Title: "Issues · " + repo.Name, Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: view})
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

func (s *Server) uiIssue(w http.ResponseWriter, r *http.Request) {
	repo, item, ok := s.uiItem(w, r, false)
	if !ok {
		return
	}
	comments, err := s.store.Comments(item)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	view := issueView{
		Item: item, Comments: comments, CanWrite: identityFrom(r).can("write", repo.Name),
		Labels: strings.Join(item.Labels, ", "), Assignees: strings.Join(item.Assignees, ", "),
	}
	s.render(w, r, http.StatusOK, "issue", page{Title: fmt.Sprintf("%s · #%d · %s", item.Title, item.Number, repo.Name), Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: view})
}

func (s *Server) uiCommentIssue(w http.ResponseWriter, r *http.Request) {
	repo, item, ok := s.uiItem(w, r, true)
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
	if _, _, err := s.store.AddComment("issue", repo.Name, item.Number, identityFrom(r).actor, body); err != nil {
		s.renderStoreError(w, r, err, "Issue not found.")
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/issues/%d", repo.Name, item.Number), http.StatusSeeOther)
}

func (s *Server) uiEditIssue(w http.ResponseWriter, r *http.Request) {
	repo, item, ok := s.uiItem(w, r, true)
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
	if _, _, err := s.store.EditItem("issue", repo.Name, item.Number, id.actor, edit); err != nil {
		s.renderStoreError(w, r, err, "Issue not found.")
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/issues/%d", repo.Name, item.Number), http.StatusSeeOther)
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
