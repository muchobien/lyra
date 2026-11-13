package server

import (
	"math"
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

	kindIssue = "issue"
	kindPull  = "pr"

	stateOpen   = "open"
	stateClosed = "closed"
	stateMerged = "merged"
	stateAll    = "all"
)

var nouns = map[string]string{kindIssue: "issue", kindPull: "pull request"}

var issueStates = []string{stateOpen, stateClosed, stateAll}

type itemWrite struct {
	store.Item
	EventID int64 `json:"event_id,omitzero"`
}

type pullWrite struct {
	store.Pull
	EventID int64    `json:"event_id,omitzero"`
	Actions []string `json:"actions,omitempty"`
}

type itemView struct {
	store.Item
	Comments []store.Comment `json:"comments"`
}

type commentWrite struct {
	store.Comment
	EventID int64 `json:"event_id"`
}

type labelWrite struct {
	store.Label
	EventID int64 `json:"event_id"`
}

type deletedWithEvent struct {
	Deleted string `json:"deleted"`
	EventID int64  `json:"event_id"`
}

type createIssueRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels"`
	Assignees []string `json:"assignees"`
}

type editItemRequest struct {
	Title           *string  `json:"title"`
	Body            *string  `json:"body"`
	State           *string  `json:"state"`
	AddLabels       []string `json:"add_labels"`
	RemoveLabels    []string `json:"remove_labels"`
	AddAssignees    []string `json:"add_assignees"`
	RemoveAssignees []string `json:"remove_assignees"`
}

type commentRequest struct {
	Body string `json:"body"`
	Path string `json:"path"`
	Line int64  `json:"line"`
	Side string `json:"side"`
}

type createLabelRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type itemQuery struct {
	state     string
	limit     int
	labels    []string
	assignees []string
}

func checkText(title *string, body *string) error {
	if title != nil {
		*title = strings.TrimSpace(*title)
		if *title == "" || utf8.RuneCountInString(*title) > maxTitle {
			return badRequest("title must be 1 to 256 characters")
		}
	}
	if body != nil && len(*body) > maxBody {
		return badRequest("body must be at most 64 KiB")
	}
	return nil
}

func itemNumber(r *http.Request) (int64, error) {
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil || number <= 0 {
		return 0, badRequest("invalid number " + r.PathValue("number"))
	}
	return number, nil
}

func parseItemQuery(r *http.Request, states []string) (itemQuery, error) {
	query := r.URL.Query()
	parsed := itemQuery{
		state:     query.Get("state"),
		limit:     queryLimit(r, 100, math.MaxInt),
		labels:    query["label"],
		assignees: identityFrom(r).resolve(query["assignee"]),
	}
	if parsed.state == "" {
		parsed.state = stateOpen
	}
	if !slices.Contains(states, parsed.state) {
		return parsed, badRequest("state must be one of " + strings.Join(states, ", "))
	}
	return parsed, nil
}

func containsAll(have, want []string) bool {
	return !slices.ContainsFunc(want, func(value string) bool {
		return !slices.Contains(have, value)
	})
}

func (query itemQuery) matches(item store.Item) bool {
	return containsAll(item.Labels, query.labels) && containsAll(item.Assignees, query.assignees)
}

func (s *Server) handleListIssues(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	query, err := parseItemQuery(r, issueStates)
	if err != nil {
		return err
	}
	items, err := s.store.Items(r.Context(), kindIssue, repo.Name, query.state)
	if err != nil {
		return err
	}
	matching := []store.Item{}
	for _, item := range items {
		if len(matching) < query.limit && query.matches(item) {
			matching = append(matching, item)
		}
	}
	return writeJSON(w, http.StatusOK, matching)
}

func (s *Server) handleCreateIssue(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return err
	}
	body, err := decodeBody[createIssueRequest](w, r)
	if err != nil {
		return err
	}
	if err := checkText(&body.Title, &body.Body); err != nil {
		return err
	}
	id := identityFrom(r)
	item, eventID, err := s.store.CreateItem(
		r.Context(),
		kindIssue,
		repo.Name,
		id.actor,
		body.Title,
		body.Body,
		body.Labels,
		id.resolve(body.Assignees),
	)
	if err != nil {
		return missing(err, "repository not found")
	}
	return writeJSON(w, http.StatusCreated, itemWrite{Item: item, EventID: eventID})
}

func (s *Server) handleGetIssue(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	number, err := itemNumber(r)
	if err != nil {
		return err
	}
	item, err := s.store.Item(ctx, kindIssue, repo.Name, number)
	if err != nil {
		return missing(err, "issue not found")
	}
	comments, err := s.store.Comments(ctx, item)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, itemView{Item: item, Comments: comments})
}

func (s *Server) handleEditItem(kind string) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		repo, err := s.lookupRepo(r, scopeWrite)
		if err != nil {
			return err
		}
		number, err := itemNumber(r)
		if err != nil {
			return err
		}
		body, err := decodeBody[editItemRequest](w, r)
		if err != nil {
			return err
		}
		if err := checkText(body.Title, body.Body); err != nil {
			return err
		}
		wasOpen := true
		if kind == kindPull {
			if wasOpen, err = s.checkReopen(ctx, repo.Name, number, body.State); err != nil {
				return err
			}
		}
		id := identityFrom(r)
		item, eventID, err := s.store.EditItem(ctx, kind, repo.Name, number, id.actor, store.ItemEdit{
			Title:           body.Title,
			Body:            body.Body,
			State:           body.State,
			AddLabels:       body.AddLabels,
			RemoveLabels:    body.RemoveLabels,
			AddAssignees:    id.resolve(body.AddAssignees),
			RemoveAssignees: id.resolve(body.RemoveAssignees),
		})
		if err != nil {
			return missing(err, nouns[kind]+" not found")
		}
		if kind == kindIssue {
			return writeJSON(w, http.StatusOK, itemWrite{Item: item, EventID: eventID})
		}
		actions := s.queueReopened(ctx, repo.Name, number, wasOpen, id.actor)
		pull, err := s.store.Pull(ctx, repo.Name, number)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, pullWrite{Pull: pull, EventID: eventID, Actions: actions})
	}
}

func (s *Server) handleCommentItem(kind string) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		repo, err := s.lookupRepo(r, scopeWrite)
		if err != nil {
			return err
		}
		number, err := itemNumber(r)
		if err != nil {
			return err
		}
		body, err := decodeBody[commentRequest](w, r)
		if err != nil {
			return err
		}
		if strings.TrimSpace(body.Body) == "" {
			return badRequest("comment body is empty")
		}
		if err := checkText(nil, &body.Body); err != nil {
			return err
		}
		anchor := store.LineRef{Path: body.Path, Line: body.Line, Side: body.Side}
		line, err := s.commentLine(ctx, repo.Name, kind, number, anchor)
		if err != nil {
			return missing(err, nouns[kind]+" not found")
		}
		actor := identityFrom(r).actor
		comment, eventID, err := s.store.AddComment(ctx, kind, repo.Name, number, actor, body.Body, line)
		if err != nil {
			return missing(err, nouns[kind]+" not found")
		}
		return writeJSON(w, http.StatusCreated, commentWrite{Comment: comment, EventID: eventID})
	}
}

func (s *Server) handleListLabels(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	labels, err := s.store.Labels(r.Context(), repo.Name)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, labels)
}

func (s *Server) handleCreateLabel(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return err
	}
	body, err := decodeBody[createLabelRequest](w, r)
	if err != nil {
		return err
	}
	actor := identityFrom(r).actor
	label, eventID, err := s.store.CreateLabel(r.Context(), repo.Name, actor, body.Name, body.Color, body.Description)
	if store.IsUniqueViolation(err) {
		return conflict("label " + body.Name + " already exists")
	}
	if err != nil {
		return missing(err, "repository not found")
	}
	return writeJSON(w, http.StatusCreated, labelWrite{Label: label, EventID: eventID})
}

func (s *Server) handleDeleteLabel(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return err
	}
	name := r.PathValue("label")
	eventID, err := s.store.DeleteLabel(r.Context(), repo.Name, identityFrom(r).actor, name)
	if err != nil {
		return missing(err, "label not found")
	}
	return writeJSON(w, http.StatusOK, deletedWithEvent{Deleted: name, EventID: eventID})
}
