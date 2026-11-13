package server

import (
	"net/http"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/muchobien/lyra/internal/store"
)

const keyAdminOnly = "key management needs an unrestricted admin token"

type createKeyRequest struct {
	Title     string `json:"title"`
	Agent     string `json:"agent"`
	PublicKey string `json:"public_key"`
	Signing   bool   `json:"signing"`
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

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) error {
	id := identityFrom(r)
	if !id.can(scopeRead, "") {
		return forbidden("listing keys needs an unrestricted token")
	}
	keys, err := s.store.Keys(r.Context(), id.actor.User)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return forbidden(keyAdminOnly)
	}
	body, err := decodeBody[createKeyRequest](w, r)
	if err != nil {
		return err
	}
	if !body.Signing {
		return badRequest("only signing keys are supported; set signing to true")
	}
	parsed, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(body.PublicKey))
	if err != nil || len(options) > 0 || strings.TrimSpace(string(rest)) != "" {
		return badRequest("public_key must be one OpenSSH public key line")
	}
	title := body.Title
	if title == "" {
		title = comment
	}
	key, err := s.store.CreateKey(ctx, store.Key{
		User:        id.actor.User,
		Title:       title,
		Agent:       body.Agent,
		PublicKey:   strings.TrimSpace(string(ssh.MarshalAuthorizedKey(parsed))),
		Fingerprint: ssh.FingerprintSHA256(parsed),
		Signing:     true,
	})
	if store.IsUniqueViolation(err) {
		return conflict("key already registered")
	}
	if err != nil {
		return err
	}
	added := keyAddedEvent{Agent: key.Agent, Fingerprint: key.Fingerprint, ID: key.ID}
	if _, err := s.store.AddEvent(ctx, "key.add", "", id.actor, added); err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, key)
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id := identityFrom(r)
	if !id.can(scopeAdmin, "") {
		return forbidden(keyAdminOnly)
	}
	keyID, err := pathID(r, "id", "invalid key id")
	if err != nil {
		return err
	}
	if err := s.store.DeleteKey(ctx, id.actor.User, keyID); err != nil {
		return missing(err, "key not found")
	}
	if _, err := s.store.AddEvent(ctx, "key.delete", "", id.actor, idPayload{ID: keyID}); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, deletedID{Deleted: keyID})
}

func (s *Server) handleSigningKey(w http.ResponseWriter, r *http.Request) error {
	publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(s.signer.PublicKey())))
	return writeJSON(w, http.StatusOK, signingKeyView{
		AllowedSigners: `lyra namespaces="git" ` + publicKey,
		Fingerprint:    ssh.FingerprintSHA256(s.signer.PublicKey()),
		PublicKey:      publicKey,
	})
}
