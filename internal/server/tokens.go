package server

import (
	"crypto/rand"
	"encoding/base32"
	"net/http"
	"strings"
	"time"

	"github.com/muchobien/lyra/internal/store"
)

const tokenAdminOnly = "token management needs an unrestricted admin token"

type createTokenRequest struct {
	Name      string   `json:"name"`
	Scope     string   `json:"scope" enum:"read,write,admin"`
	Repos     []string `json:"repos,omitzero" doc:"owner/repo or owner/*; empty means unrestricted"`
	ExpiresIn string   `json:"expires_in,omitzero" doc:"A Go duration, such as 720h"`
}

type tokenCreatedEvent struct {
	ID    int64    `json:"id"`
	Name  string   `json:"name"`
	Repos []string `json:"repos"`
	Scope string   `json:"scope"`
}

type tokenCreated struct {
	Secret string      `json:"secret"`
	Token  store.Token `json:"token"`
}

type idPayload struct {
	ID int64 `json:"id"`
}

type revokedID struct {
	Revoked int64 `json:"revoked"`
}

type deletedID struct {
	Deleted int64 `json:"deleted"`
}

func (s *Server) handleListTokens(r *http.Request, _ *struct{}) (*body[[]store.Token], error) {
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return nil, forbidden(tokenAdminOnly)
	}
	tokens, err := s.store.Tokens(r.Context(), id.actor.User)
	if err != nil {
		return nil, err
	}
	return reply(tokens)
}

func tokenExpiry(expiresIn string) (*time.Time, error) {
	if expiresIn == "" {
		return nil, nil
	}
	duration, err := time.ParseDuration(expiresIn)
	if err != nil || duration <= 0 {
		return nil, badRequest("expires_in must be a positive Go duration such as 720h")
	}
	expiresAt := time.Now().Add(duration)
	return &expiresAt, nil
}

func (s *Server) handleCreateToken(r *http.Request, input *bodyInput[createTokenRequest]) (*body[tokenCreated], error) {
	ctx := r.Context()
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return nil, forbidden(tokenAdminOnly)
	}
	request := input.Body
	if strings.TrimSpace(request.Name) == "" {
		return nil, badRequest("name is required")
	}
	if request.Repos == nil {
		request.Repos = []string{}
	}
	expiresAt, err := tokenExpiry(request.ExpiresIn)
	if err != nil {
		return nil, err
	}
	random := make([]byte, 32)
	rand.Read(random)
	secret := "lyra_" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(random))
	token, err := s.store.CreateToken(ctx, request.Name, id.actor.User, secret, request.Scope, request.Repos, expiresAt)
	if err != nil {
		return nil, err
	}
	created := tokenCreatedEvent{ID: token.ID, Name: token.Name, Repos: token.Repos, Scope: token.Scope}
	if _, err := s.store.AddEvent(ctx, "token.create", "", id.actor, created); err != nil {
		return nil, err
	}
	return reply(tokenCreated{Secret: secret, Token: token})
}

func (s *Server) handleDeleteToken(r *http.Request, input *idPath) (*body[revokedID], error) {
	ctx := r.Context()
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return nil, forbidden(tokenAdminOnly)
	}
	if err := s.store.DeleteToken(ctx, id.actor.User, input.ID); err != nil {
		return nil, missing(err, "token not found")
	}
	if _, err := s.store.AddEvent(ctx, "token.revoke", "", id.actor, idPayload{ID: input.ID}); err != nil {
		return nil, err
	}
	return reply(revokedID{Revoked: input.ID})
}
