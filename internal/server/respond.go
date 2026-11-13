package server

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/muchobien/lyra/internal/store"
)

const maxRequestBody = 1 << 20

type statusError struct {
	status  int
	message string
}

func (failure statusError) Error() string {
	return failure.message
}

func badRequest(message string) error {
	return statusError{status: http.StatusBadRequest, message: message}
}

func forbidden(message string) error {
	return statusError{status: http.StatusForbidden, message: message}
}

func notFound(message string) error {
	return statusError{status: http.StatusNotFound, message: message}
}

func conflict(message string) error {
	return statusError{status: http.StatusConflict, message: message}
}

func missing(err error, message string) error {
	if errors.Is(err, store.ErrNotFound) {
		return notFound(message)
	}
	return err
}

func invalidAsConflict(err error) error {
	if invalid, ok := errors.AsType[store.InvalidError](err); ok {
		return conflict(invalid.Error())
	}
	return err
}

func errorStatus(err error) (int, string) {
	if failure, ok := errors.AsType[statusError](err); ok {
		return failure.status, failure.message
	}
	if invalid, ok := errors.AsType[store.InvalidError](err); ok {
		return http.StatusBadRequest, invalid.Error()
	}
	if errors.Is(err, store.ErrNotFound) {
		return http.StatusNotFound, "not found"
	}
	return http.StatusInternalServerError, err.Error()
}

type handler func(w http.ResponseWriter, r *http.Request) error

func detached(r *http.Request) *http.Request {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return r
	}
	return r.WithContext(context.WithoutCancel(r.Context()))
}

func api(handle handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handle(w, detached(r)); err != nil {
			status, message := errorStatus(err)
			writeError(w, status, message)
		}
	}
}

func (s *Server) page(handle handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handle(w, detached(r)); err != nil {
			status, message := errorStatus(err)
			s.renderError(w, r, status, message)
		}
	}
}

type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) error {
	encoded, err := json.Marshal(value, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(append(encoded, '\n'))
	return nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	if err := writeJSON(w, status, errorBody{Error: message}); err != nil {
		slog.Error("write error response", "status", status, "error", err)
	}
}

func decodeBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var body T
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	if err := json.UnmarshalRead(reader, &body, json.RejectUnknownMembers(true)); err != nil {
		return body, badRequest("invalid JSON body: " + err.Error())
	}
	return body, nil
}

func pathID(r *http.Request, name, problem string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0, badRequest(problem)
	}
	return id, nil
}

func queryLimit(r *http.Request, fallback, most int) int {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 || limit > most {
		return fallback
	}
	return limit
}
