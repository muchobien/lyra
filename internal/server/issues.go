package server

import (
	"math"
	"net/http"
	"slices"
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
	Body      string   `json:"body,omitzero"`
	Labels    []string `json:"labels,omitzero" doc:"Labels must exist."`
	Assignees []string `json:"assignees,omitzero" doc:"Logins or agent names; @me is the caller."`
}

type editItemRequest struct {
	Title           *string  `json:"title,omitzero"`
	Body            *string  `json:"body,omitzero"`
	State           *string  `json:"state,omitzero" enum:"open,closed"`
	AddLabels       []string `json:"add_labels,omitzero"`
	RemoveLabels    []string `json:"remove_labels,omitzero"`
	AddAssignees    []string `json:"add_assignees,omitzero"`
	RemoveAssignees []string `json:"remove_assignees,omitzero"`
}

type commentRequest struct {
	Body string `json:"body"`
	Path string `json:"path,omitzero" doc:"Pull requests only: a file in the diff."`
	Line int64  `json:"line,omitzero" doc:"Pull requests only: a line of path in the diff."`
	Side string `json:"side,omitzero" enum:"new,old" doc:"new is the head version, old the base version."`
}

type createLabelRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color,omitzero" doc:"Hex without #, such as d73a4a."`
	Description string `json:"description,omitzero"`
}

type ItemPath struct {
	RepoPath
	Number int64 `path:"number" minimum:"1"`
}

type listItemsInput struct {
	RepoPath
	State    string   `query:"state" doc:"open (default), closed or all; merged too for pull requests."`
	Label    []string `query:"label,explode" doc:"Items carrying every one of these labels."`
	Assignee []string `query:"assignee,explode" doc:"Items assigned to every one of these names; @me is the caller."`
	Limit    int      `query:"limit" doc:"Defaults to 100."`
}

type labelInput struct {
	RepoPath
	Label string `path:"label"`
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

func parseItemQuery(r *http.Request, input *listItemsInput, states []string) (itemQuery, error) {
	parsed := itemQuery{
		state:     input.State,
		limit:     queryLimit(input.Limit, 100, math.MaxInt),
		labels:    input.Label,
		assignees: identityFrom(r).resolve(input.Assignee),
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

func (s *Server) handleListIssues(r *http.Request, input *listItemsInput) (*body[[]store.Item], error) {
	repo, err := s.lookupRepo(r, scopeRead, input.fullName())
	if err != nil {
		return nil, err
	}
	query, err := parseItemQuery(r, input, issueStates)
	if err != nil {
		return nil, err
	}
	items, err := s.store.Items(r.Context(), kindIssue, repo.Name, query.state)
	if err != nil {
		return nil, err
	}
	matching := []store.Item{}
	for _, item := range items {
		if len(matching) < query.limit && query.matches(item) {
			matching = append(matching, item)
		}
	}
	return reply(matching)
}

type createIssueInput struct {
	RepoPath
	Body createIssueRequest
}

func (s *Server) handleCreateIssue(r *http.Request, input *createIssueInput) (*body[itemWrite], error) {
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return nil, err
	}
	request := input.Body
	if err := checkText(&request.Title, &request.Body); err != nil {
		return nil, err
	}
	id := identityFrom(r)
	item, eventID, err := s.store.CreateItem(
		r.Context(),
		kindIssue,
		repo.Name,
		id.actor,
		request.Title,
		request.Body,
		request.Labels,
		id.resolve(request.Assignees),
	)
	if err != nil {
		return nil, missing(err, "repository not found")
	}
	return reply(itemWrite{Item: item, EventID: eventID})
}

func (s *Server) handleGetIssue(r *http.Request, input *ItemPath) (*body[itemView], error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeRead, input.fullName())
	if err != nil {
		return nil, err
	}
	item, err := s.store.Item(ctx, kindIssue, repo.Name, input.Number)
	if err != nil {
		return nil, missing(err, "issue not found")
	}
	comments, err := s.store.Comments(ctx, item)
	if err != nil {
		return nil, err
	}
	return reply(itemView{Item: item, Comments: comments})
}

type editItemInput struct {
	ItemPath
	Body editItemRequest
}

type itemEdited struct {
	repo    string
	item    store.Item
	eventID int64
	actions []string
}

func (s *Server) editItem(r *http.Request, kind string, input *editItemInput) (itemEdited, error) {
	ctx := r.Context()
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return itemEdited{}, err
	}
	request := input.Body
	if err := checkText(request.Title, request.Body); err != nil {
		return itemEdited{}, err
	}
	wasOpen := true
	if kind == kindPull {
		if wasOpen, err = s.checkReopen(ctx, repo.Name, input.Number, request.State); err != nil {
			return itemEdited{}, err
		}
	}
	id := identityFrom(r)
	item, eventID, err := s.store.EditItem(ctx, kind, repo.Name, input.Number, id.actor, store.ItemEdit{
		Title:           request.Title,
		Body:            request.Body,
		State:           request.State,
		AddLabels:       request.AddLabels,
		RemoveLabels:    request.RemoveLabels,
		AddAssignees:    id.resolve(request.AddAssignees),
		RemoveAssignees: id.resolve(request.RemoveAssignees),
	})
	if err != nil {
		return itemEdited{}, missing(err, nouns[kind]+" not found")
	}
	edited := itemEdited{repo: repo.Name, item: item, eventID: eventID}
	if kind == kindPull {
		edited.actions = s.queueReopened(ctx, repo.Name, input.Number, wasOpen, id.actor)
	}
	return edited, nil
}

func (s *Server) handleEditIssue(r *http.Request, input *editItemInput) (*body[itemWrite], error) {
	edited, err := s.editItem(r, kindIssue, input)
	if err != nil {
		return nil, err
	}
	return reply(itemWrite{Item: edited.item, EventID: edited.eventID})
}

type commentInput struct {
	ItemPath
	Body commentRequest
}

func (s *Server) handleCommentItem(kind string) func(*http.Request, *commentInput) (*body[commentWrite], error) {
	return func(r *http.Request, input *commentInput) (*body[commentWrite], error) {
		ctx := r.Context()
		repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
		if err != nil {
			return nil, err
		}
		request := input.Body
		if strings.TrimSpace(request.Body) == "" {
			return nil, badRequest("comment body is empty")
		}
		if err := checkText(nil, &request.Body); err != nil {
			return nil, err
		}
		anchor := store.LineRef{Path: request.Path, Line: request.Line, Side: request.Side}
		line, err := s.commentLine(ctx, repo.Name, kind, input.Number, anchor)
		if err != nil {
			return nil, missing(err, nouns[kind]+" not found")
		}
		actor := identityFrom(r).actor
		comment, eventID, err := s.store.AddComment(ctx, kind, repo.Name, input.Number, actor, request.Body, line)
		if err != nil {
			return nil, missing(err, nouns[kind]+" not found")
		}
		return reply(commentWrite{Comment: comment, EventID: eventID})
	}
}

func (s *Server) handleListLabels(r *http.Request, input *RepoPath) (*body[[]store.Label], error) {
	repo, err := s.lookupRepo(r, scopeRead, input.fullName())
	if err != nil {
		return nil, err
	}
	labels, err := s.store.Labels(r.Context(), repo.Name)
	if err != nil {
		return nil, err
	}
	return reply(labels)
}

type createLabelInput struct {
	RepoPath
	Body createLabelRequest
}

func (s *Server) handleCreateLabel(r *http.Request, input *createLabelInput) (*body[labelWrite], error) {
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return nil, err
	}
	request := input.Body
	actor := identityFrom(r).actor
	label, eventID, err := s.store.CreateLabel(
		r.Context(), repo.Name, actor, request.Name, request.Color, request.Description,
	)
	if store.IsUniqueViolation(err) {
		return nil, conflict("label " + request.Name + " already exists")
	}
	if err != nil {
		return nil, missing(err, "repository not found")
	}
	return reply(labelWrite{Label: label, EventID: eventID})
}

func (s *Server) handleDeleteLabel(r *http.Request, input *labelInput) (*body[deletedWithEvent], error) {
	repo, err := s.lookupRepo(r, scopeWrite, input.fullName())
	if err != nil {
		return nil, err
	}
	eventID, err := s.store.DeleteLabel(r.Context(), repo.Name, identityFrom(r).actor, input.Label)
	if err != nil {
		return nil, missing(err, "label not found")
	}
	return reply(deletedWithEvent{Deleted: input.Label, EventID: eventID})
}
