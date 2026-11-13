package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/muchobien/lyra/internal/store"
)

const (
	runnerOnline       = time.Minute
	runnerAdminOnly    = "runner management needs an unrestricted admin token"
	maxRunnerName      = 255
	maxRunnerTokenName = 100
)

type nameRequest struct {
	Name string `json:"name"`
}

type runnerView struct {
	store.Runner
	Online bool `json:"online"`
}

type runnerCreated struct {
	store.Runner
	Token string `json:"token"`
	URL   string `json:"url"`
}

type runnerTokenCreated struct {
	store.RunnerToken
	Secret  string `json:"secret"`
	URL     string `json:"url"`
	EventID int64  `json:"event_id"`
}

type deletedIDWithEvent struct {
	Deleted int64 `json:"deleted"`
	EventID int64 `json:"event_id"`
}

func newUUID() string {
	value := randomHex(16)
	return value[0:8] + "-" + value[8:12] + "-" + value[12:16] + "-" + value[16:20] + "-" + value[20:32]
}

func randomHex(size int) string {
	secret := make([]byte, size)
	rand.Read(secret)
	return hex.EncodeToString(secret)
}

func (s *Server) actionsURL(r *http.Request) string {
	if s.config.ActionsURL != "" {
		return s.config.ActionsURL
	}
	return baseURL(r)
}

func runnerAdmin(r *http.Request) error {
	if !identityFrom(r).can(scopeAdmin, "") {
		return forbidden(runnerAdminOnly)
	}
	return nil
}

func runnerPathID(r *http.Request) (int64, error) {
	return pathID(r, "id", "invalid id "+r.PathValue("id"))
}

func requestedName(w http.ResponseWriter, r *http.Request, longest int) (string, error) {
	body, err := decodeBody[nameRequest](w, r)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || utf8.RuneCountInString(name) > longest {
		return "", badRequest(fmt.Sprintf("name must be 1 to %d characters", longest))
	}
	return name, nil
}

func (s *Server) handleListRunners(w http.ResponseWriter, r *http.Request) error {
	if err := runnerAdmin(r); err != nil {
		return err
	}
	runners, err := s.store.Runners(r.Context())
	if err != nil {
		return err
	}
	views := []runnerView{}
	for _, runner := range runners {
		online := runner.LastSeenAt != nil && time.Since(*runner.LastSeenAt) < runnerOnline
		views = append(views, runnerView{Runner: runner, Online: online})
	}
	return writeJSON(w, http.StatusOK, views)
}

func (s *Server) handleCreateRunner(w http.ResponseWriter, r *http.Request) error {
	if err := runnerAdmin(r); err != nil {
		return err
	}
	name, err := requestedName(w, r, maxRunnerName)
	if err != nil {
		return err
	}
	secret := randomHex(20)
	registered := store.Runner{UUID: newUUID(), Name: name}
	runner, err := s.store.CreateRunner(r.Context(), identityFrom(r).actor, registered, secret, 0)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, runnerCreated{Runner: runner, Token: secret, URL: s.actionsURL(r)})
}

func (s *Server) handleDeleteRunner(w http.ResponseWriter, r *http.Request) error {
	if err := runnerAdmin(r); err != nil {
		return err
	}
	id, err := runnerPathID(r)
	if err != nil {
		return err
	}
	eventID, err := s.store.DeleteRunner(r.Context(), identityFrom(r).actor, id)
	if err != nil {
		return missing(err, fmt.Sprintf("runner %d not found", id))
	}
	return writeJSON(w, http.StatusOK, deletedIDWithEvent{Deleted: id, EventID: eventID})
}

func (s *Server) handleListRunnerTokens(w http.ResponseWriter, r *http.Request) error {
	if err := runnerAdmin(r); err != nil {
		return err
	}
	tokens, err := s.store.RunnerTokens(r.Context())
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, tokens)
}

func (s *Server) handleCreateRunnerToken(w http.ResponseWriter, r *http.Request) error {
	if err := runnerAdmin(r); err != nil {
		return err
	}
	name, err := requestedName(w, r, maxRunnerTokenName)
	if err != nil {
		return err
	}
	secret := randomHex(20)
	token, eventID, err := s.store.CreateRunnerToken(r.Context(), identityFrom(r).actor, name, secret)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, runnerTokenCreated{
		RunnerToken: token,
		Secret:      secret,
		URL:         s.actionsURL(r),
		EventID:     eventID,
	})
}

func (s *Server) handleDeleteRunnerToken(w http.ResponseWriter, r *http.Request) error {
	if err := runnerAdmin(r); err != nil {
		return err
	}
	id, err := runnerPathID(r)
	if err != nil {
		return err
	}
	eventID, err := s.store.DeleteRunnerToken(r.Context(), identityFrom(r).actor, id)
	if err != nil {
		return missing(err, fmt.Sprintf("runner token %d not found", id))
	}
	return writeJSON(w, http.StatusOK, deletedIDWithEvent{Deleted: id, EventID: eventID})
}
