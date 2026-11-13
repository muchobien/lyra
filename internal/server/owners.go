package server

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/muchobien/lyra/internal/store"
)

type ownerGroup struct {
	store.Owner
	Repos []store.Repo `json:"repos"`
}

func (id *identity) seesOwner(owner string) bool {
	return len(id.repos) == 0 || slices.ContainsFunc(id.repos, func(repo string) bool { return strings.HasPrefix(repo, owner+"/") })
}

func (s *Server) ownerGroups(id *identity, only string) ([]ownerGroup, error) {
	owners, err := s.store.Owners()
	if err != nil {
		return nil, err
	}
	repos, err := s.store.Repos()
	if err != nil {
		return nil, err
	}
	groups := []ownerGroup{}
	for _, owner := range owners {
		if (only != "" && owner.Name != only) || !id.seesOwner(owner.Name) {
			continue
		}
		group := ownerGroup{Owner: owner, Repos: []store.Repo{}}
		for _, repo := range repos {
			if strings.HasPrefix(repo.Name, owner.Name+"/") && id.can("read", repo.Name) {
				group.Repos = append(group.Repos, repo)
			}
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (s *Server) handleListOwners(w http.ResponseWriter, r *http.Request) {
	groups, err := s.ownerGroups(identityFrom(r), "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (s *Server) handleGetOwner(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("owner")
	groups, err := s.ownerGroups(identityFrom(r), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(groups) == 0 {
		writeError(w, http.StatusNotFound, "owner "+name+" not found")
		return
	}
	writeJSON(w, http.StatusOK, groups[0])
}

func (s *Server) handleCreateOwner(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("admin", "") {
		writeError(w, http.StatusForbidden, "owner management needs an unrestricted admin token")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if !repoNamePattern.MatchString(body.Name) || slices.Contains(reservedNames, body.Name) {
		writeError(w, http.StatusBadRequest, "owner names are 1 to 100 letters, digits, ., _ or -, starting with a letter or digit, and not api, static, healthz or v2")
		return
	}
	if utf8.RuneCountInString(body.Description) > maxTitle {
		writeError(w, http.StatusBadRequest, "description must be at most 256 characters")
		return
	}
	owner, eventID, err := s.store.CreateOwner(id.actor, body.Name, body.Description)
	if errors.Is(err, store.ErrExists) {
		writeError(w, http.StatusConflict, "owner "+body.Name+" already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		store.Owner
		EventID int64 `json:"event_id"`
	}{owner, eventID})
}

func (s *Server) handleDeleteOwner(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("admin", "") {
		writeError(w, http.StatusForbidden, "owner management needs an unrestricted admin token")
		return
	}
	name := r.PathValue("owner")
	eventID, err := s.store.DeleteOwner(id.actor, name)
	var invalid store.InvalidError
	if errors.As(err, &invalid) {
		writeError(w, http.StatusConflict, invalid.Error())
		return
	}
	if err != nil {
		writeStoreError(w, err, "owner "+name+" not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": name, "event_id": eventID})
}

func (s *Server) uiOwner(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("owner")
	groups, err := s.ownerGroups(identityFrom(r), name)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if len(groups) == 0 {
		s.renderError(w, r, http.StatusNotFound, "Owner "+name+" not found.")
		return
	}
	s.render(w, r, http.StatusOK, "repos", page{Title: name, Data: groups})
}
