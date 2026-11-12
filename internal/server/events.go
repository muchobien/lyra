package server

import (
	"encoding/json"
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

const eventBatch = 100

func sameActor(first, second store.Actor) bool {
	if first.Session != "" || second.Session != "" {
		return first.User == second.User && first.Agent == second.Agent && first.Session == second.Session
	}
	return first.User == second.User && first.Via == second.Via && first.Agent == second.Agent
}

func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	query := r.URL.Query()
	repo := query.Get("repo")
	if repo != "" && !id.can("read", repo) {
		writeError(w, http.StatusForbidden, "token lacks read access to "+repo)
		return
	}
	actions := query["action"]
	for _, pattern := range actions {
		if _, err := path.Match(pattern, ""); err != nil {
			writeError(w, http.StatusBadRequest, "invalid action pattern "+pattern)
			return
		}
	}
	var item int64
	if value := query.Get("item"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 || repo == "" {
			writeError(w, http.StatusBadRequest, "item must be an issue or pull request number, with repo")
			return
		}
		item = parsed
	}
	excludeSelf := query.Get("exclude_self") == "true"
	assignee := query.Get("assignee")
	if assignee == "me" || assignee == "@me" {
		assignee = id.name()
	}
	start := r.Header.Get("Last-Event-ID")
	if start == "" {
		start = query.Get("since")
	}
	cursor, err := s.store.LatestEventID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	switch {
	case start == "":
	case start == "mine":
		if item == 0 {
			writeError(w, http.StatusBadRequest, "since=mine needs repo and item")
			return
		}
		events, err := s.store.ItemEvents(repo, item)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, event := range events {
			if sameActor(event.Actor, id.actor) {
				cursor = event.ID
			}
		}
		cursors, err := s.store.WaitCursors(repo, item)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, delivered := range cursors {
			if sameActor(delivered.Actor, id.actor) {
				cursor = max(cursor, delivered.EventID)
			}
		}
	default:
		if cursor, err = strconv.ParseInt(start, 10, 64); err != nil || cursor < 0 {
			writeError(w, http.StatusBadRequest, "invalid event id "+start)
			return
		}
	}
	matches := func(event store.Event) bool {
		if !id.can("read", event.Repo) || (repo != "" && event.Repo != repo) || (item != 0 && event.Item != item) {
			return false
		}
		if excludeSelf && sameActor(event.Actor, id.actor) {
			return false
		}
		if assignee != "" {
			var data struct {
				Assignee string `json:"assignee"`
			}
			if !strings.HasSuffix(event.Action, ".assigned") || json.Unmarshal(event.Data, &data) != nil || data.Assignee != assignee {
				return false
			}
		}
		return len(actions) == 0 || slices.ContainsFunc(actions, func(pattern string) bool {
			matched, _ := path.Match(pattern, event.Action)
			return matched
		})
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	controller := http.NewResponseController(w)
	fmt.Fprintf(w, "id: %d\n\n", cursor)
	if controller.Flush() != nil {
		return
	}
	poll := time.NewTicker(time.Second)
	defer poll.Stop()
	keepalive := time.NewTicker(25 * time.Second)
	defer keepalive.Stop()
	for {
		changed := s.store.Changed()
		events, err := s.store.EventsAfter(cursor, eventBatch)
		if err != nil {
			slog.Error("event stream", "err", err)
			return
		}
		var delivered int64
		for _, event := range events {
			cursor = event.ID
			if !matches(event) {
				continue
			}
			encoded, err := json.Marshal(event)
			if err != nil {
				slog.Error("event stream", "err", err)
				return
			}
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Action, encoded)
			delivered = event.ID
		}
		if len(events) > 0 && controller.Flush() != nil {
			return
		}
		if delivered != 0 && item != 0 {
			if err := s.store.SetWaitCursor(repo, item, id.actor, delivered); err != nil {
				slog.Error("event stream", "err", err)
				return
			}
		}
		if len(events) == eventBatch {
			continue
		}
		select {
		case <-r.Context().Done():
			return
		case <-changed:
		case <-poll.C:
		case <-keepalive.C:
			if id.tokenID != 0 {
				if active, err := s.store.TokenActive(id.tokenID); err != nil || !active {
					return
				}
			}
			fmt.Fprintf(w, ": keepalive\nid: %d\n\n", cursor)
			if controller.Flush() != nil {
				return
			}
		}
	}
}
