package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	v2 "github.com/distribution/distribution/v3/registry/api/v2"
	"github.com/gorilla/mux"

	"github.com/muchobien/lyra/internal/store"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
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
	json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]string{{"code": code, "message": message}}})
}

func (s *Server) imageProblem(id *identity, name, scope string) (int, string, string) {
	repo, _, _ := strings.Cut(name, "/")
	if !id.can(scope, repo) {
		return http.StatusForbidden, "DENIED", "token lacks " + scope + " access to " + repo
	}
	if _, err := s.store.Repo(repo); errors.Is(err, store.ErrNotFound) {
		return http.StatusNotFound, "NAME_UNKNOWN", "no repository " + repo + "; image names start with a Lyra repository"
	} else if err != nil {
		return http.StatusInternalServerError, "UNKNOWN", err.Error()
	}
	return 0, "", ""
}

func (s *Server) handleRegistry(w http.ResponseWriter, r *http.Request) {
	var match mux.RouteMatch
	if !v2.Router().Match(r, &match) || match.Route == nil {
		s.registry.ServeHTTP(w, r)
		return
	}
	id := identityFrom(r)
	route := match.Route.GetName()
	name := match.Vars["name"]
	switch {
	case route == v2.RouteNameCatalog:
		if len(id.repos) > 0 {
			registryError(w, http.StatusForbidden, "DENIED", "the catalog needs a token that isn't limited to some repositories")
			return
		}
	case name != "":
		scope := "write"
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			scope = "read"
		case http.MethodDelete:
			scope = "admin"
		}
		if status, code, message := s.imageProblem(id, name, scope); status != 0 {
			registryError(w, status, code, message)
			return
		}
		query := r.URL.Query()
		if from := query.Get("from"); from != "" {
			if status, _, _ := s.imageProblem(id, from, "read"); status != 0 {
				query.Del("from")
				query.Del("mount")
				r.URL.RawQuery = query.Encode()
			}
		}
	}
	if route != v2.RouteNameManifest || (r.Method != http.MethodPut && r.Method != http.MethodDelete) {
		s.registry.ServeHTTP(w, r)
		return
	}
	recorder := &statusRecorder{ResponseWriter: w}
	s.registry.ServeHTTP(recorder, r)
	repo, _, _ := strings.Cut(name, "/")
	reference := match.Vars["reference"]
	byDigest := strings.Contains(reference, ":")
	var err error
	switch {
	case r.Method == http.MethodPut && recorder.status == http.StatusCreated && !byDigest:
		_, _, err = s.store.SetImageTag(id.actor, store.Image{
			Repo: repo, Name: name, Tag: reference, Digest: recorder.Header().Get("Docker-Content-Digest"), MediaType: r.Header.Get("Content-Type"),
		})
	case r.Method == http.MethodDelete && recorder.status == http.StatusAccepted:
		_, err = s.store.DeleteImage(id.actor, repo, name, reference, byDigest)
	}
	if err != nil {
		slog.Error("record image", "name", name, "reference", reference, "error", err)
	}
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

func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	images, err := s.store.Images(repo.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for index := range images {
		images[index].Reference = s.imageReference(r, images[index])
	}
	writeJSON(w, http.StatusOK, images)
}

func (s *Server) uiImages(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.uiLookup(w, r)
	if !ok {
		return
	}
	images, err := s.store.Images(repo.Name)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	for index := range images {
		images[index].Reference = s.imageReference(r, images[index])
	}
	view := struct {
		Images []store.Image
		Host   string
	}{images, s.registryHost(r)}
	s.render(w, r, http.StatusOK, "images", page{Title: "Images · " + repo.Name, Repo: &repo, Ref: s.git.DefaultBranch(s.repoDir(repo.Name)), CloneURL: cloneURL(r, repo.Name), Data: view})
}
