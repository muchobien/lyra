package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strconv"
	"time"

	"github.com/muchobien/lyra/internal/store"
)

const eventBatch = 100

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
	start := r.Header.Get("Last-Event-ID")
	if start == "" {
		start = query.Get("since")
	}
	var cursor int64
	var err error
	if start == "" {
		cursor, err = s.store.LatestEventID()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if cursor, err = strconv.ParseInt(start, 10, 64); err != nil || cursor < 0 {
		writeError(w, http.StatusBadRequest, "invalid event id "+start)
		return
	}
	matches := func(event store.Event) bool {
		if !id.can("read", event.Repo) || (repo != "" && event.Repo != repo) {
			return false
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
		}
		if len(events) > 0 && controller.Flush() != nil {
			return
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
