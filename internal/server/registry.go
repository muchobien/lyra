package server

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	v2 "github.com/distribution/distribution/v3/registry/api/v2"
	"github.com/gorilla/mux"

	"github.com/muchobien/lyra/internal/store"
)

const (
	registryRepositories = "/docker/registry/v2/repositories/"

	registryDenied      = "DENIED"
	registryNameUnknown = "NAME_UNKNOWN"
	registryUnknown     = "UNKNOWN"
	registryUnavailable = "UNAVAILABLE"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

type registryProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	status  int
}

type registryErrors struct {
	Errors []registryProblem `json:"errors"`
}

type imagesView struct {
	Images []store.Image
	Host   string
}

func (recorder *statusRecorder) WriteHeader(status int) {
	if recorder.status == 0 {
		recorder.status = status
	}
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(data []byte) (int, error) {
	if recorder.status == 0 {
		recorder.status = http.StatusOK
	}
	return recorder.ResponseWriter.Write(data)
}

func (recorder *statusRecorder) Unwrap() http.ResponseWriter {
	return recorder.ResponseWriter
}

func registryError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(status)
	problems := registryErrors{Errors: []registryProblem{{Code: code, Message: message}}}
	if encoded, err := json.Marshal(problems); err == nil {
		w.Write(append(encoded, '\n'))
	}
}

func imageRepo(name string) string {
	parts := strings.SplitN(name, "/", 3)
	if len(parts) < 2 {
		return name
	}
	return parts[0] + "/" + parts[1]
}

func imageScope(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead:
		return scopeRead
	case http.MethodDelete:
		return scopeAdmin
	}
	return scopeWrite
}

func (s *Server) imageProblem(ctx context.Context, id *identity, name, scope string) *registryProblem {
	repo := imageRepo(name)
	if !id.can(scope, repo) {
		return &registryProblem{
			status:  http.StatusForbidden,
			Code:    registryDenied,
			Message: "token lacks " + scope + " access to " + repo,
		}
	}
	_, err := s.store.Repo(ctx, repo)
	if errors.Is(err, store.ErrNotFound) {
		return &registryProblem{
			status:  http.StatusNotFound,
			Code:    registryNameUnknown,
			Message: "no repository " + repo + "; image names start with a repository's owner/name",
		}
	}
	if err != nil {
		return &registryProblem{status: http.StatusInternalServerError, Code: registryUnknown, Message: err.Error()}
	}
	return nil
}

func (s *Server) authorizeImage(r *http.Request, route, name string) *registryProblem {
	id := identityFrom(r)
	if route == v2.RouteNameCatalog && len(id.repos) > 0 {
		return &registryProblem{
			status:  http.StatusForbidden,
			Code:    registryDenied,
			Message: "the catalog needs a token that isn't limited to some repositories",
		}
	}
	if route == v2.RouteNameCatalog || name == "" {
		return nil
	}
	if problem := s.imageProblem(r.Context(), id, name, imageScope(r.Method)); problem != nil {
		return problem
	}
	query := r.URL.Query()
	from := query.Get("from")
	if from != "" && s.imageProblem(r.Context(), id, from, scopeRead) != nil {
		query.Del("from")
		query.Del("mount")
		r.URL.RawQuery = query.Encode()
	}
	return nil
}

func (s *Server) recordManifest(r *http.Request, recorder *statusRecorder, name, reference string) {
	ctx := context.WithoutCancel(r.Context())
	actor := identityFrom(r).actor
	repo := imageRepo(name)
	byDigest := strings.Contains(reference, ":")
	var err error
	switch {
	case r.Method == http.MethodPut && recorder.status == http.StatusCreated && !byDigest:
		_, _, err = s.store.SetImageTag(ctx, actor, store.Image{
			Repo:      repo,
			Name:      name,
			Tag:       reference,
			Digest:    recorder.Header().Get("Docker-Content-Digest"),
			MediaType: r.Header.Get("Content-Type"),
		})
	case r.Method == http.MethodDelete && recorder.status == http.StatusAccepted:
		_, err = s.store.DeleteImage(ctx, actor, repo, name, reference, byDigest)
	}
	if err != nil {
		slog.Error("record image", "name", name, "reference", reference, "error", err)
	}
}

func (s *Server) handleRegistry(w http.ResponseWriter, r *http.Request) {
	var match mux.RouteMatch
	if !v2.Router().Match(r, &match) || match.Route == nil {
		s.registry.ServeHTTP(w, r)
		return
	}
	route := match.Route.GetName()
	name := match.Vars["name"]
	if problem := s.authorizeImage(r, route, name); problem != nil {
		registryError(w, problem.status, problem.Code, problem.Message)
		return
	}
	changesManifest := route == v2.RouteNameManifest && (r.Method == http.MethodPut || r.Method == http.MethodDelete)
	if !changesManifest {
		s.registry.ServeHTTP(w, r)
		return
	}
	recorder := &statusRecorder{ResponseWriter: w}
	s.registry.ServeHTTP(recorder, r)
	s.recordManifest(r, recorder, name, match.Vars["reference"])
}

func (s *Server) registryHost(r *http.Request) string {
	if s.config.RegistryHost != "" {
		return s.config.RegistryHost
	}
	return r.Host
}

func (s *Server) imageReference(r *http.Request, image store.Image) string {
	return s.registryHost(r) + "/" + image.Name + ":" + image.Tag
}

func (s *Server) images(r *http.Request, repo string) ([]store.Image, error) {
	images, err := s.store.Images(r.Context(), repo)
	for index := range images {
		images[index].Reference = s.imageReference(r, images[index])
	}
	return images, err
}

func (s *Server) handleListImages(r *http.Request, _ *RepoPath) (*body[[]store.Image], error) {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return nil, err
	}
	images, err := s.images(r, repo.Name)
	if err != nil {
		return nil, err
	}
	return reply(images)
}

func (s *Server) uiImages(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.uiLookup(r)
	if err != nil {
		return err
	}
	images, err := s.images(r, repo.Name)
	if err != nil {
		return err
	}
	view := imagesView{Images: images, Host: s.registryHost(r)}
	ref := s.git.DefaultBranch(r.Context(), s.repoDir(repo.Name))
	s.render(w, r, http.StatusOK, "images", s.repoPage(r, &repo, "Images · "+repo.Name, ref, view))
	return nil
}
