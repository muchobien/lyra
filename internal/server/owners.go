package server

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/muchobien/lyra/internal/store"
)

const (
	ownerAdminOnly   = "owner management needs an unrestricted admin token"
	ownerNameProblem = "owner names are 1 to 100 letters, digits, ., _ or -, starting with a letter or digit, " +
		"and not api, static, healthz or v2"
)

var repoNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)

var reservedNames = []string{"api", "static", "healthz", "v2"}

type ownerGroup struct {
	store.Owner
	Repos []store.Repo `json:"repos"`
}

type createOwnerRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ownerWrite struct {
	store.Owner
	EventID int64 `json:"event_id"`
}

func (s *Server) ownerGroups(ctx context.Context, id *identity, only string) ([]ownerGroup, error) {
	owners, err := s.store.Owners(ctx)
	if err != nil {
		return nil, err
	}
	repos, err := s.store.Repos(ctx)
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
			if strings.HasPrefix(repo.Name, owner.Name+"/") && id.can(scopeRead, repo.Name) {
				group.Repos = append(group.Repos, repo)
			}
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (s *Server) handleListOwners(w http.ResponseWriter, r *http.Request) error {
	groups, err := s.ownerGroups(r.Context(), identityFrom(r), "")
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, groups)
}

func (s *Server) handleGetOwner(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("owner")
	groups, err := s.ownerGroups(r.Context(), identityFrom(r), name)
	if err != nil {
		return err
	}
	if len(groups) == 0 {
		return notFound("owner " + name + " not found")
	}
	return writeJSON(w, http.StatusOK, groups[0])
}

func (s *Server) handleCreateOwner(w http.ResponseWriter, r *http.Request) error {
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return forbidden(ownerAdminOnly)
	}
	body, err := decodeBody[createOwnerRequest](w, r)
	if err != nil {
		return err
	}
	if !repoNamePattern.MatchString(body.Name) || slices.Contains(reservedNames, body.Name) {
		return badRequest(ownerNameProblem)
	}
	if utf8.RuneCountInString(body.Description) > maxTitle {
		return badRequest("description must be at most 256 characters")
	}
	owner, eventID, err := s.store.CreateOwner(r.Context(), id.actor, body.Name, body.Description)
	if errors.Is(err, store.ErrExists) {
		return conflict("owner " + body.Name + " already exists")
	}
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, ownerWrite{Owner: owner, EventID: eventID})
}

func (s *Server) handleDeleteOwner(w http.ResponseWriter, r *http.Request) error {
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return forbidden(ownerAdminOnly)
	}
	name := r.PathValue("owner")
	eventID, err := s.store.DeleteOwner(r.Context(), id.actor, name)
	if err != nil {
		return missing(invalidAsConflict(err), "owner "+name+" not found")
	}
	return writeJSON(w, http.StatusOK, deletedWithEvent{Deleted: name, EventID: eventID})
}
