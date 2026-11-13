package server

import (
	"net/http"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/muchobien/lyra/internal/store"
)

const keyAdminOnly = "key management needs an unrestricted admin token"

type createKeyRequest struct {
	Title     string `json:"title,omitzero"`
	Agent     string `json:"agent,omitzero"`
	PublicKey string `json:"public_key" doc:"One OpenSSH public key line"`
	Signing   bool   `json:"signing" doc:"Must be true: only signing keys are supported"`
}

type keyAddedEvent struct {
	Agent       string `json:"agent"`
	Fingerprint string `json:"fingerprint"`
	ID          int64  `json:"id"`
}

type signingKeyView struct {
	AllowedSigners string `json:"allowed_signers"`
	Fingerprint    string `json:"fingerprint"`
	PublicKey      string `json:"public_key"`
}

func (s *Server) handleListKeys(r *http.Request, _ *struct{}) (*body[[]store.Key], error) {
	id := identityFrom(r)
	if !id.can(scopeRead, "") {
		return nil, forbidden("listing keys needs an unrestricted token")
	}
	keys, err := s.store.Keys(r.Context(), id.actor.User)
	if err != nil {
		return nil, err
	}
	return reply(keys)
}

func (s *Server) handleCreateKey(r *http.Request, input *bodyInput[createKeyRequest]) (*body[store.Key], error) {
	ctx := r.Context()
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return nil, forbidden(keyAdminOnly)
	}
	request := input.Body
	if !request.Signing {
		return nil, badRequest("only signing keys are supported; set signing to true")
	}
	parsed, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(request.PublicKey))
	if err != nil || len(options) > 0 || strings.TrimSpace(string(rest)) != "" {
		return nil, badRequest("public_key must be one OpenSSH public key line")
	}
	title := request.Title
	if title == "" {
		title = comment
	}
	key, err := s.store.CreateKey(ctx, store.Key{
		User:        id.actor.User,
		Title:       title,
		Agent:       request.Agent,
		PublicKey:   strings.TrimSpace(string(ssh.MarshalAuthorizedKey(parsed))),
		Fingerprint: ssh.FingerprintSHA256(parsed),
		Signing:     true,
	})
	if store.IsUniqueViolation(err) {
		return nil, conflict("key already registered")
	}
	if err != nil {
		return nil, err
	}
	added := keyAddedEvent{Agent: key.Agent, Fingerprint: key.Fingerprint, ID: key.ID}
	if _, err := s.store.AddEvent(ctx, "key.add", "", id.actor, added); err != nil {
		return nil, err
	}
	return reply(key)
}

func (s *Server) handleDeleteKey(r *http.Request, input *idPath) (*body[deletedID], error) {
	ctx := r.Context()
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return nil, forbidden(keyAdminOnly)
	}
	if err := s.store.DeleteKey(ctx, id.actor.User, input.ID); err != nil {
		return nil, missing(err, "key not found")
	}
	if _, err := s.store.AddEvent(ctx, "key.delete", "", id.actor, idPayload{ID: input.ID}); err != nil {
		return nil, err
	}
	return reply(deletedID{Deleted: input.ID})
}

func (s *Server) handleSigningKey(_ *http.Request, _ *struct{}) (*body[signingKeyView], error) {
	publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(s.signer.PublicKey())))
	return reply(signingKeyView{
		AllowedSigners: `lyra namespaces="git" ` + publicKey,
		Fingerprint:    ssh.FingerprintSHA256(s.signer.PublicKey()),
		PublicKey:      publicKey,
	})
}
