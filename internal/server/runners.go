package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	pingv1 "code.forgejo.org/forgejo/actions-proto/ping/v1"
	runnerv1 "code.forgejo.org/forgejo/actions-proto/runner/v1"
	"code.forgejo.org/forgejo/actions-proto/runner/v1/runnerv1connect"
	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"

	"github.com/muchobien/lyra/internal/store"
)

const runnerOnline = time.Minute

type runnerService struct {
	runnerv1connect.UnimplementedRunnerServiceHandler
	server *Server
}

type pingService struct{}

func (pingService) Ping(ctx context.Context, request *connect.Request[pingv1.PingRequest]) (*connect.Response[pingv1.PingResponse], error) {
	return connect.NewResponse(&pingv1.PingResponse{Data: "Hello, " + request.Msg.Data + "!"}), nil
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

func runnerMessage(runner store.Runner) *runnerv1.Runner {
	return &runnerv1.Runner{
		Id: runner.ID, Uuid: runner.UUID, Name: runner.Name, Version: runner.Version, Labels: runner.Labels,
		Ephemeral: runner.Ephemeral, Status: runnerv1.RunnerStatus_RUNNER_STATUS_IDLE,
	}
}

func (service runnerService) authenticate(header http.Header) (store.Runner, error) {
	runner, err := service.server.store.RunnerByUUID(header.Get("x-runner-uuid"))
	if errors.Is(err, store.ErrNotFound) {
		return runner, connect.NewError(connect.CodeUnauthenticated, errors.New("unregistered runner"))
	}
	if err != nil {
		return runner, connect.NewError(connect.CodeInternal, err)
	}
	if subtle.ConstantTimeCompare([]byte(store.HashToken(header.Get("x-runner-token"))), []byte(runner.TokenHash)) != 1 {
		return runner, connect.NewError(connect.CodeUnauthenticated, errors.New("unregistered runner"))
	}
	return runner, nil
}

func (service runnerService) Register(ctx context.Context, request *connect.Request[runnerv1.RegisterRequest]) (*connect.Response[runnerv1.RegisterResponse], error) {
	registration, err := service.server.store.RunnerTokenBySecret(strings.TrimSpace(request.Msg.Token))
	if errors.Is(err, store.ErrNotFound) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unknown registration token"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	name := strings.TrimSpace(request.Msg.Name)
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("runner name must be 1 to 255 characters"))
	}
	uuid := newUUID()
	secret := randomHex(20)
	actor := store.Actor{User: registration.User, Via: "runner:" + name}
	runner, err := service.server.store.CreateRunner(actor, store.Runner{
		UUID: uuid, Name: name, Version: request.Msg.Version, Ephemeral: request.Msg.Ephemeral,
	}, secret, registration.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	message := runnerMessage(runner)
	message.Token = secret
	return connect.NewResponse(&runnerv1.RegisterResponse{Runner: message}), nil
}

func (service runnerService) Declare(ctx context.Context, request *connect.Request[runnerv1.DeclareRequest]) (*connect.Response[runnerv1.DeclareResponse], error) {
	runner, err := service.authenticate(request.Header())
	if err != nil {
		return nil, err
	}
	if runner, err = service.server.store.DeclareRunner(runner.ID, request.Msg.Labels, request.Msg.Version); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&runnerv1.DeclareResponse{Runner: runnerMessage(runner)}), nil
}

func (service runnerService) FetchTask(ctx context.Context, request *connect.Request[runnerv1.FetchTaskRequest]) (*connect.Response[runnerv1.FetchTaskResponse], error) {
	runner, err := service.authenticate(request.Header())
	if err != nil {
		return nil, err
	}
	if err := service.server.store.TouchRunner(runner.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	task, err := service.task(runner)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&runnerv1.FetchTaskResponse{Task: task, TasksVersion: request.Msg.TasksVersion}), nil
}

func (service runnerService) FetchSingleTask(ctx context.Context, request *connect.Request[runnerv1.FetchSingleTaskRequest]) (*connect.Response[runnerv1.FetchSingleTaskResponse], error) {
	runner, err := service.authenticate(request.Header())
	if err != nil {
		return nil, err
	}
	if err := service.server.store.TouchRunner(runner.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	task, err := service.task(runner)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&runnerv1.FetchSingleTaskResponse{Task: task, TasksVersion: request.Msg.TasksVersion}), nil
}

type runnerView struct {
	store.Runner
	Online bool `json:"online"`
}

func requireAccountAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !identityFrom(r).can("admin", "") {
		writeError(w, http.StatusForbidden, "runner management needs an unrestricted admin token")
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id "+chi.URLParam(r, "id"))
		return 0, false
	}
	return id, true
}

func (s *Server) handleListRunners(w http.ResponseWriter, r *http.Request) {
	if !requireAccountAdmin(w, r) {
		return
	}
	runners, err := s.store.Runners()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	views := []runnerView{}
	for _, runner := range runners {
		views = append(views, runnerView{runner, runner.LastSeenAt != nil && time.Since(*runner.LastSeenAt) < runnerOnline})
	}
	writeJSON(w, http.StatusOK, views)
}

func (s *Server) handleDeleteRunner(w http.ResponseWriter, r *http.Request) {
	if !requireAccountAdmin(w, r) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	eventID, err := s.store.DeleteRunner(identityFrom(r).actor, id)
	if err != nil {
		writeStoreError(w, err, fmt.Sprintf("runner %d not found", id))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": id, "event_id": eventID})
}

func (s *Server) handleListRunnerTokens(w http.ResponseWriter, r *http.Request) {
	if !requireAccountAdmin(w, r) {
		return
	}
	tokens, err := s.store.RunnerTokens()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (s *Server) handleCreateRunnerToken(w http.ResponseWriter, r *http.Request) {
	if !requireAccountAdmin(w, r) {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.Name = strings.TrimSpace(body.Name); body.Name == "" || utf8.RuneCountInString(body.Name) > 100 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 100 characters")
		return
	}
	secret := randomHex(20)
	token, eventID, err := s.store.CreateRunnerToken(identityFrom(r).actor, body.Name, secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		store.RunnerToken
		Secret  string `json:"secret"`
		URL     string `json:"url"`
		EventID int64  `json:"event_id"`
	}{token, secret, s.actionsURL(r), eventID})
}

func (s *Server) handleDeleteRunnerToken(w http.ResponseWriter, r *http.Request) {
	if !requireAccountAdmin(w, r) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	eventID, err := s.store.DeleteRunnerToken(identityFrom(r).actor, id)
	if err != nil {
		writeStoreError(w, err, fmt.Sprintf("runner token %d not found", id))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": id, "event_id": eventID})
}

func (s *Server) actionsURL(r *http.Request) string {
	if s.config.ActionsURL != "" {
		return s.config.ActionsURL
	}
	return baseURL(r)
}

func (s *Server) handleCreateRunner(w http.ResponseWriter, r *http.Request) {
	if !requireAccountAdmin(w, r) {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.Name = strings.TrimSpace(body.Name); body.Name == "" || utf8.RuneCountInString(body.Name) > 255 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 255 characters")
		return
	}
	uuid := newUUID()
	secret := randomHex(20)
	runner, err := s.store.CreateRunner(identityFrom(r).actor, store.Runner{UUID: uuid, Name: body.Name}, secret, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		store.Runner
		Token string `json:"token"`
		URL   string `json:"url"`
	}{runner, secret, s.actionsURL(r)})
}
