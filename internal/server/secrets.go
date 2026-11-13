package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var secretNamePattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,99}$`)

const maxSecret = 64 << 10

func loadSecretsKey(config Config) ([]byte, error) {
	file := config.SecretsKeyFile
	if file == "" {
		file = filepath.Join(config.DataDir, "secrets_key")
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			key := make([]byte, 32)
			rand.Read(key)
			if err := os.WriteFile(file, []byte(hex.EncodeToString(key)+"\n"), 0o600); err != nil {
				return nil, err
			}
			slog.Warn("generated a secrets key in the data directory; it exists only there and in backups, so keep it with --secrets-key-file", "file", file)
		}
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("secrets key: %w", err)
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(content)))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("secrets key %s: want 64 hex characters (32 bytes)", file)
	}
	return key, nil
}

func (s *Server) sealSecret(plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.secretsKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, plain, nil), nil
}

func (s *Server) openSecret(sealed []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.secretsKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(sealed) < aead.NonceSize() {
		return nil, errors.New("secret too short")
	}
	return aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
}

func (s *Server) repoSecrets(repo string) (map[string]string, error) {
	stored, err := s.store.Secrets(repo)
	if err != nil {
		return nil, err
	}
	secrets := map[string]string{}
	for _, secret := range stored {
		plain, err := s.openSecret(secret.Value)
		if err != nil {
			return nil, fmt.Errorf("secret %s: %w", secret.Name, err)
		}
		secrets[secret.Name] = string(plain)
	}
	return secrets, nil
}

func secretName(w http.ResponseWriter, r *http.Request) (string, bool) {
	name := strings.ToUpper(r.PathValue("name"))
	if !secretNamePattern.MatchString(name) || strings.HasPrefix(name, "GITHUB_") || strings.HasPrefix(name, "FORGEJO_") || strings.HasPrefix(name, "GITEA_") {
		writeError(w, http.StatusBadRequest, "names are letters, digits and _, not starting with a digit, GITHUB_, FORGEJO_ or GITEA_")
		return "", false
	}
	return name, true
}

func (s *Server) handleListSecrets(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "admin")
	if !ok {
		return
	}
	secrets, err := s.store.Secrets(repo.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, secrets)
}

func (s *Server) handleSetSecret(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "admin")
	if !ok {
		return
	}
	name, ok := secretName(w, r)
	if !ok {
		return
	}
	var body struct {
		Value string `json:"value"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.Value == "" || len(body.Value) > maxSecret {
		writeError(w, http.StatusBadRequest, "a secret is 1 byte to 64 KiB")
		return
	}
	sealed, err := s.sealSecret([]byte(body.Value))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	eventID, err := s.store.SetSecret(identityFrom(r).actor, repo.Name, name, sealed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "event_id": eventID})
}

func (s *Server) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "admin")
	if !ok {
		return
	}
	name, ok := secretName(w, r)
	if !ok {
		return
	}
	eventID, err := s.store.DeleteSecret(identityFrom(r).actor, repo.Name, name)
	if err != nil {
		writeStoreError(w, err, "no secret "+name)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": name, "event_id": eventID})
}

func (s *Server) handleListVariables(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	variables, err := s.store.Variables(repo.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, variables)
}

func (s *Server) handleSetVariable(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	name, ok := secretName(w, r)
	if !ok {
		return
	}
	var body struct {
		Value string `json:"value"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if len(body.Value) > maxSecret {
		writeError(w, http.StatusBadRequest, "a variable is at most 64 KiB")
		return
	}
	eventID, err := s.store.SetVariable(identityFrom(r).actor, repo.Name, name, body.Value)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "value": body.Value, "event_id": eventID})
}

func (s *Server) handleDeleteVariable(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "write")
	if !ok {
		return
	}
	name, ok := secretName(w, r)
	if !ok {
		return
	}
	eventID, err := s.store.DeleteVariable(identityFrom(r).actor, repo.Name, name)
	if err != nil {
		writeStoreError(w, err, "no variable "+name)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": name, "event_id": eventID})
}
