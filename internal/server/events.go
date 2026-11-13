package server

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/muchobien/lyra/internal/store"
)

const (
	eventBatch        = 100
	streamPoll        = time.Second
	streamKeepalive   = 25 * time.Second
	sinceOwnLastEvent = "mine"
)

type eventFilter struct {
	id          *identity
	repo        string
	item        int64
	actions     []string
	assignee    string
	excludeSelf bool
}

type assignment struct {
	Assignee string `json:"assignee"`
}

func sameActor(first, second store.Actor) bool {
	if first.Session != "" || second.Session != "" {
		return first.User == second.User && first.Agent == second.Agent && first.Session == second.Session
	}
	return first.User == second.User && first.Via == second.Via && first.Agent == second.Agent
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) error {
	repo := r.URL.Query().Get("repo")
	if !identityFrom(r).can(scopeRead, repo) {
		return forbidden("token lacks read access")
	}
	events, err := s.store.Events(r.Context(), repo, queryLimit(r, 50, 1000))
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, events)
}

func parseEventFilter(r *http.Request) (eventFilter, error) {
	query := r.URL.Query()
	filter := eventFilter{
		id:          identityFrom(r),
		repo:        query.Get("repo"),
		actions:     query["action"],
		assignee:    query.Get("assignee"),
		excludeSelf: query.Get("exclude_self") == "true",
	}
	if filter.repo != "" && !filter.id.can(scopeRead, filter.repo) {
		return filter, forbidden("token lacks read access to " + filter.repo)
	}
	for _, pattern := range filter.actions {
		if _, err := path.Match(pattern, ""); err != nil {
			return filter, badRequest("invalid action pattern " + pattern)
		}
	}
	if value := query.Get("item"); value != "" {
		item, err := strconv.ParseInt(value, 10, 64)
		if err != nil || item <= 0 || filter.repo == "" {
			return filter, badRequest("item must be an issue or pull request number, with repo")
		}
		filter.item = item
	}
	if filter.assignee == "me" || filter.assignee == "@me" {
		filter.assignee = filter.id.name()
	}
	return filter, nil
}

func (filter eventFilter) assigns(event store.Event) bool {
	var data assignment
	return strings.HasSuffix(event.Action, ".assigned") &&
		json.Unmarshal(event.Data, &data) == nil &&
		data.Assignee == filter.assignee
}

func (filter eventFilter) matches(event store.Event) bool {
	if !filter.id.can(scopeRead, event.Repo) {
		return false
	}
	if filter.repo != "" && event.Repo != filter.repo {
		return false
	}
	if filter.item != 0 && event.Item != filter.item {
		return false
	}
	if filter.excludeSelf && sameActor(event.Actor, filter.id.actor) {
		return false
	}
	if filter.assignee != "" && !filter.assigns(event) {
		return false
	}
	return len(filter.actions) == 0 || slices.ContainsFunc(filter.actions, func(pattern string) bool {
		matched, _ := path.Match(pattern, event.Action)
		return matched
	})
}

func (s *Server) ownCursor(ctx context.Context, filter eventFilter, latest int64) (int64, error) {
	if filter.item == 0 {
		return 0, badRequest("since=mine needs repo and item")
	}
	cursor := latest
	events, err := s.store.ItemEvents(ctx, filter.repo, filter.item)
	if err != nil {
		return 0, err
	}
	for _, event := range events {
		if sameActor(event.Actor, filter.id.actor) {
			cursor = event.ID
		}
	}
	cursors, err := s.store.WaitCursors(ctx, filter.repo, filter.item)
	if err != nil {
		return 0, err
	}
	for _, delivered := range cursors {
		if sameActor(delivered.Actor, filter.id.actor) {
			cursor = max(cursor, delivered.EventID)
		}
	}
	return cursor, nil
}

func (s *Server) streamStart(r *http.Request, filter eventFilter) (int64, error) {
	start := r.Header.Get("Last-Event-ID")
	if start == "" {
		start = r.URL.Query().Get("since")
	}
	latest, err := s.store.LatestEventID(r.Context())
	if err != nil {
		return 0, err
	}
	switch start {
	case "":
		return latest, nil
	case sinceOwnLastEvent:
		return s.ownCursor(r.Context(), filter, latest)
	}
	cursor, err := strconv.ParseInt(start, 10, 64)
	if err != nil || cursor < 0 {
		return 0, badRequest("invalid event id " + start)
	}
	return cursor, nil
}

func writeEvents(w http.ResponseWriter, filter eventFilter, events []store.Event) (int64, error) {
	var delivered int64
	for _, event := range events {
		if !filter.matches(event) {
			continue
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			return delivered, err
		}
		fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Action, encoded)
		delivered = event.ID
	}
	return delivered, nil
}

func (s *Server) tokenStillActive(ctx context.Context, id *identity) bool {
	if id.tokenID == 0 {
		return true
	}
	active, err := s.store.TokenActive(ctx, id.tokenID)
	return err == nil && active
}

func (s *Server) streamEvents(w http.ResponseWriter, r *http.Request, filter eventFilter, cursor int64) error {
	ctx := r.Context()
	controller := http.NewResponseController(w)
	poll := time.NewTicker(streamPoll)
	defer poll.Stop()
	keepalive := time.NewTicker(streamKeepalive)
	defer keepalive.Stop()
	for {
		changed := s.store.Changed()
		events, err := s.store.EventsAfter(ctx, cursor, eventBatch)
		if err != nil {
			return err
		}
		delivered, err := writeEvents(w, filter, events)
		if err != nil {
			return err
		}
		if len(events) > 0 {
			cursor = events[len(events)-1].ID
			if controller.Flush() != nil {
				return nil
			}
		}
		if delivered != 0 && filter.item != 0 {
			saved := context.WithoutCancel(ctx)
			if err := s.store.SetWaitCursor(saved, filter.repo, filter.item, filter.id.actor, delivered); err != nil {
				return err
			}
		}
		if len(events) == eventBatch {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-changed:
		case <-poll.C:
		case <-keepalive.C:
			if !s.tokenStillActive(ctx, filter.id) {
				return nil
			}
			fmt.Fprintf(w, ": keepalive\nid: %d\n\n", cursor)
			if controller.Flush() != nil {
				return nil
			}
		}
	}
}

func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) error {
	filter, err := parseEventFilter(r)
	if err != nil {
		return err
	}
	cursor, err := s.streamStart(r, filter)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "id: %d\n\n", cursor)
	if http.NewResponseController(w).Flush() != nil {
		return nil
	}
	if err := s.streamEvents(w, r, filter, cursor); err != nil && r.Context().Err() == nil {
		slog.Error("event stream", "err", err)
	}
	return nil
}
