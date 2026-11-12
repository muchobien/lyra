package server

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/muchobien/lyra/internal/store"
)

const (
	maxTitle = 256
	maxBody  = 65536
)

var nouns = map[string]string{"issue": "issue", "pr": "pull request"}

type itemWrite struct {
	store.Item
	EventID int64 `json:"event_id,omitempty"`
}

type pullWrite struct {
	store.Pull
	EventID int64 `json:"event_id,omitempty"`
}

type itemView struct {
	store.Item
	Comments []store.Comment `json:"comments"`
}

type itemQuery struct {
	state     string
	limit     int
	labels    []string
	assignees []string
}

func (id *identity) name() string {
	if id.actor.Agent != "" {
		return id.actor.Agent
	}
	return id.actor.User
}

func (id *identity) resolve(names []string) []string {
	resolved := []string{}
	for _, name := range names {
		if name == "@me" {
			name = id.name()
		}
		resolved = append(resolved, name)
	}
	return resolved
}

func checkText(title *string, body *string) string {
	if title != nil {
		*title = strings.TrimSpace(*title)
		if *title == "" || utf8.RuneCountInString(*title) > maxTitle {
			return "title must be 1 to 256 characters"
		}
	}
	if body != nil && len(*body) > maxBody {
		return "body must be at most 64 KiB"
	}
	return ""
}

func writeStoreError(w http.ResponseWriter, err error, missing string) {
	var invalid store.InvalidError
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, missing)
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func itemNumber(w http.ResponseWriter, r *http.Request) (int64, bool) {
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil || number <= 0 {
		writeError(w, http.StatusBadRequest, "invalid number "+r.PathValue("number"))
		return 0, false
	}
	return number, true
}

func parseItemQuery(w http.ResponseWriter, r *http.Request, states []string) (itemQuery, bool) {
	query := r.URL.Query()
	parsed := itemQuery{state: query.Get("state"), labels: query["label"], assignees: identityFrom(r).resolve(query["assignee"])}
	if parsed.state == "" {
		parsed.state = "open"
	}
	if !slices.Contains(states, parsed.state) {
		writeError(w, http.StatusBadRequest, "state must be one of "+strings.Join(states, ", "))
		return parsed, false
	}
	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit <= 0 {
		limit = 100
	}
	parsed.limit = limit
	return parsed, true
}

func (query itemQuery) matches(item store.Item) bool {
	hasLabels := !slices.ContainsFunc(query.labels, func(label string) bool { return !slices.Contains(item.Labels, label) })
	hasAssignees := !slices.ContainsFunc(query.assignees, func(assignee string) bool { return !slices.Contains(item.Assignees, assignee) })
	return hasLabels && hasAssignees
}

func (s *Server) handleListIssues(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	query, ok := parseItemQuery(w, r, []string{"open", "closed", "all"})
	if !ok {
		return
	}
	items, err := s.store.Items("issue", repo.Name, query.state)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	matching := []store.Item{}
	for _, item := range items {
		if len(matching) < query.limit && query.matches(item) {
			matching = append(matching, item)
		}
	}
	writeJSON(w, http.StatusOK, matching)
}

func (s *Server) handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	var body struct {
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		Labels    []string `json:"labels"`
		Assignees []string `json:"assignees"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if problem := checkText(&body.Title, &body.Body); problem != "" {
		writeError(w, http.StatusBadRequest, problem)
		return
	}
	id := identityFrom(r)
	item, eventID, err := s.store.CreateItem("issue", repo.Name, id.actor, body.Title, body.Body, body.Labels, id.resolve(body.Assignees))
	if err != nil {
		writeStoreError(w, err, "repository not found")
		return
	}
	writeJSON(w, http.StatusCreated, itemWrite{Item: item, EventID: eventID})
}

func (s *Server) handleGetIssue(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	number, ok := itemNumber(w, r)
	if !ok {
		return
	}
	item, err := s.store.Item("issue", repo.Name, number)
	if err != nil {
		writeStoreError(w, err, "issue not found")
		return
	}
	comments, err := s.store.Comments(item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, itemView{Item: item, Comments: comments})
}

func (s *Server) handleEditItem(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := s.lookupRepo(w, r, "write")
		if !ok {
			return
		}
		number, ok := itemNumber(w, r)
		if !ok {
			return
		}
		var body struct {
			Title           *string  `json:"title"`
			Body            *string  `json:"body"`
			State           *string  `json:"state"`
			AddLabels       []string `json:"add_labels"`
			RemoveLabels    []string `json:"remove_labels"`
			AddAssignees    []string `json:"add_assignees"`
			RemoveAssignees []string `json:"remove_assignees"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		if problem := checkText(body.Title, body.Body); problem != "" {
			writeError(w, http.StatusBadRequest, problem)
			return
		}
		id := identityFrom(r)
		item, eventID, err := s.store.EditItem(kind, repo.Name, number, id.actor, store.ItemEdit{
			Title: body.Title, Body: body.Body, State: body.State,
			AddLabels: body.AddLabels, RemoveLabels: body.RemoveLabels,
			AddAssignees: id.resolve(body.AddAssignees), RemoveAssignees: id.resolve(body.RemoveAssignees),
		})
		if err != nil {
			writeStoreError(w, err, nouns[kind]+" not found")
			return
		}
		if kind == "issue" {
			writeJSON(w, http.StatusOK, itemWrite{Item: item, EventID: eventID})
			return
		}
		pull, err := s.store.Pull(repo.Name, number)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, pullWrite{Pull: pull, EventID: eventID})
	}
}

func (s *Server) handleCommentItem(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo, ok := s.lookupRepo(w, r, "write")
		if !ok {
			return
		}
		number, ok := itemNumber(w, r)
		if !ok {
			return
		}
		var body struct {
			Body string `json:"body"`
			Path string `json:"path"`
			Line int64  `json:"line"`
			Side string `json:"side"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		if strings.TrimSpace(body.Body) == "" {
			writeError(w, http.StatusBadRequest, "comment body is empty")
			return
		}
		if problem := checkText(nil, &body.Body); problem != "" {
			writeError(w, http.StatusBadRequest, problem)
			return
		}
		line, err := s.commentLine(repo.Name, kind, number, body.Path, body.Line, body.Side)
		if err != nil {
			writeStoreError(w, err, nouns[kind]+" not found")
			return
		}
		comment, eventID, err := s.store.AddComment(kind, repo.Name, number, identityFrom(r).actor, body.Body, line)
		if err != nil {
			writeStoreError(w, err, nouns[kind]+" not found")
			return
		}
		writeJSON(w, http.StatusCreated, struct {
			store.Comment
			EventID int64 `json:"event_id"`
		}{comment, eventID})
	}
}

func (s *Server) handleListLabels(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	labels, err := s.store.Labels(repo.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, labels)
}

func (s *Server) handleCreateLabel(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	label, eventID, err := s.store.CreateLabel(repo.Name, identityFrom(r).actor, body.Name, body.Color, body.Description)
	if store.IsUniqueViolation(err) {
		writeError(w, http.StatusConflict, "label "+body.Name+" already exists")
		return
	}
	if err != nil {
		writeStoreError(w, err, "repository not found")
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		store.Label
		EventID int64 `json:"event_id"`
	}{label, eventID})
}

func (s *Server) handleDeleteLabel(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	name := r.PathValue("label")
	eventID, err := s.store.DeleteLabel(repo.Name, identityFrom(r).actor, name)
	if err != nil {
		writeStoreError(w, err, "label not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": name, "event_id": eventID})
}
