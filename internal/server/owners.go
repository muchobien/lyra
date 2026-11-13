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
	Description string `json:"description,omitzero"`
}

type ownerPath struct {
	Owner string `path:"owner"`
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

func (s *Server) handleListOwners(r *http.Request, _ *struct{}) (*body[[]ownerGroup], error) {
	groups, err := s.ownerGroups(r.Context(), identityFrom(r), "")
	if err != nil {
		return nil, err
	}
	return reply(groups)
}

func (s *Server) handleGetOwner(r *http.Request, input *ownerPath) (*body[ownerGroup], error) {
	groups, err := s.ownerGroups(r.Context(), identityFrom(r), input.Owner)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, notFound("owner " + input.Owner + " not found")
	}
	return reply(groups[0])
}

func (s *Server) handleCreateOwner(r *http.Request, input *bodyInput[createOwnerRequest]) (*body[ownerWrite], error) {
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return nil, forbidden(ownerAdminOnly)
	}
	request := input.Body
	if !repoNamePattern.MatchString(request.Name) || slices.Contains(reservedNames, request.Name) {
		return nil, badRequest(ownerNameProblem)
	}
	if utf8.RuneCountInString(request.Description) > maxTitle {
		return nil, badRequest("description must be at most 256 characters")
	}
	owner, eventID, err := s.store.CreateOwner(r.Context(), id.actor, request.Name, request.Description)
	if errors.Is(err, store.ErrExists) {
		return nil, conflict("owner " + request.Name + " already exists")
	}
	if err != nil {
		return nil, err
	}
	return reply(ownerWrite{Owner: owner, EventID: eventID})
}

func (s *Server) handleDeleteOwner(r *http.Request, input *ownerPath) (*body[deletedWithEvent], error) {
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return nil, forbidden(ownerAdminOnly)
	}
	eventID, err := s.store.DeleteOwner(r.Context(), id.actor, input.Owner)
	if err != nil {
		return nil, missing(invalidAsConflict(err), "owner "+input.Owner+" not found")
	}
	return reply(deletedWithEvent{Deleted: input.Owner, EventID: eventID})
}

func (s *Server) uiOwner(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("owner")
	groups, err := s.ownerGroups(r.Context(), identityFrom(r), name)
	if err != nil {
		return err
	}
	if len(groups) == 0 {
		return notFound("Owner " + name + " not found.")
	}
	s.render(w, r, http.StatusOK, "repos", page{Title: name, Data: groups})
	return nil
}
