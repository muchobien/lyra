package server

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"

	"github.com/muchobien/lyra/internal/store"
)

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
	Message string `json:"error"`
	status  int
}

func (body *errorBody) Error() string {
	return body.Message
}

func (body *errorBody) GetStatus() int {
	return body.status
}

func newAPIError(status int, message string, errs ...error) huma.StatusError {
	details := []string{}
	for _, err := range errs {
		if detail, ok := errors.AsType[*huma.ErrorDetail](err); ok {
			details = append(details, detail.Message+" ("+detail.Location+")")
		} else if err != nil {
			details = append(details, err.Error())
		}
	}
	if len(details) > 0 {
		message += ": " + strings.Join(details, "; ")
	}
	return &errorBody{Message: message, status: status}
}

func encodeJSON(w io.Writer, value any) error {
	encoded, err := json.Marshal(value, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return err
	}
	_, err = w.Write(append(encoded, '\n'))
	return err
}

var jsonFormat = huma.Format{
	Marshal: encodeJSON,
	Unmarshal: func(data []byte, value any) error {
		return json.Unmarshal(data, value, json.RejectUnknownMembers(true))
	},
}

func writeJSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return encodeJSON(w, value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	if err := writeJSON(w, status, errorBody{Message: message}); err != nil {
		slog.Error("write error response", "status", status, "error", err)
	}
}

type requestKey struct{}

func withRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = detached(r)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestKey{}, r)))
	})
}

type body[T any] struct {
	Body T
}

func reply[T any](value T) (*body[T], error) {
	return &body[T]{Body: value}, nil
}

func register[I, O any](api huma.API, operation huma.Operation, handle func(*http.Request, *I) (*O, error)) {
	huma.Register(api, operation, func(ctx context.Context, input *I) (*O, error) {
		output, err := handle(ctx.Value(requestKey{}).(*http.Request), input)
		if err != nil {
			status, message := errorStatus(err)
			return nil, &errorBody{Message: message, status: status}
		}
		return output, nil
	})
}

func stream[I any](
	handle func(http.ResponseWriter, *http.Request, *I) error,
) func(*http.Request, *I) (*huma.StreamResponse, error) {
	return func(_ *http.Request, input *I) (*huma.StreamResponse, error) {
		return &huma.StreamResponse{Body: func(ctx huma.Context) {
			r, w := humachi.Unwrap(ctx)
			api(func(w http.ResponseWriter, r *http.Request) error {
				return handle(w, r, input)
			})(w, r)
		}}, nil
	}
}

func queryLimit(limit, fallback, most int) int {
	if limit <= 0 || limit > most {
		return fallback
	}
	return limit
}

type RepoPath struct {
	Owner string `path:"owner"`
	Repo  string `path:"repo"`
}

func (path RepoPath) fullName() string {
	return path.Owner + "/" + path.Repo
}

type idPath struct {
	ID int64 `path:"id"`
}

type bodyInput[T any] struct {
	Body T
}
