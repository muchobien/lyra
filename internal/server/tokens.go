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
	Scope     string   `json:"scope"`
	Repos     []string `json:"repos"`
	ExpiresIn string   `json:"expires_in"`
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

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) error {
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return forbidden(tokenAdminOnly)
	}
	tokens, err := s.store.Tokens(r.Context(), id.actor.User)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, tokens)
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

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return forbidden(tokenAdminOnly)
	}
	body, err := decodeBody[createTokenRequest](w, r)
	if err != nil {
		return err
	}
	if strings.TrimSpace(body.Name) == "" {
		return badRequest("name is required")
	}
	if _, ok := scopeRank[body.Scope]; !ok {
		return badRequest("scope must be read, write or admin")
	}
	if body.Repos == nil {
		body.Repos = []string{}
	}
	expiresAt, err := tokenExpiry(body.ExpiresIn)
	if err != nil {
		return err
	}
	random := make([]byte, 32)
	rand.Read(random)
	secret := "lyra_" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(random))
	token, err := s.store.CreateToken(ctx, body.Name, id.actor.User, secret, body.Scope, body.Repos, expiresAt)
	if err != nil {
		return err
	}
	created := tokenCreatedEvent{ID: token.ID, Name: token.Name, Repos: token.Repos, Scope: token.Scope}
	if _, err := s.store.AddEvent(ctx, "token.create", "", id.actor, created); err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, tokenCreated{Secret: secret, Token: token})
}

func (s *Server) handleDeleteToken(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return forbidden(tokenAdminOnly)
	}
	tokenID, err := pathID(r, "id", "invalid token id")
	if err != nil {
		return err
	}
	if err := s.store.DeleteToken(ctx, id.actor.User, tokenID); err != nil {
		return missing(err, "token not found")
	}
	if _, err := s.store.AddEvent(ctx, "token.revoke", "", id.actor, idPayload{ID: tokenID}); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, revokedID{Revoked: tokenID})
}
