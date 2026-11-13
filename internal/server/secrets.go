package server

import (
	"context"
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
	"slices"
	"strings"
)

const (
	maxSecret         = 64 << 10
	secretNameProblem = "names are letters, digits and _, not starting with a digit, GITHUB_, FORGEJO_ or GITEA_"
)

var secretNamePattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,99}$`)

var reservedSecretPrefixes = []string{"GITHUB_", "FORGEJO_", "GITEA_"}

type valueRequest struct {
	Value string `json:"value"`
}

type secretSet struct {
	EventID int64  `json:"event_id"`
	Name    string `json:"name"`
}

type variableSet struct {
	EventID int64  `json:"event_id"`
	Name    string `json:"name"`
	Value   string `json:"value"`
}

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
			slog.Warn(
				"generated a secrets key in the data directory; it exists only there and in backups, "+
					"so keep it with --secrets-key-file",
				"file", file,
			)
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

func (s *Server) secretCipher() (cipher.AEAD, error) {
	block, err := aes.NewCipher(s.secretsKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (s *Server) sealSecret(plain []byte) ([]byte, error) {
	aead, err := s.secretCipher()
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
	aead, err := s.secretCipher()
	if err != nil {
		return nil, err
	}
	if len(sealed) < aead.NonceSize() {
		return nil, errors.New("secret too short")
	}
	return aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
}

func (s *Server) repoSecrets(ctx context.Context, repo string) (map[string]string, error) {
	stored, err := s.store.Secrets(ctx, repo)
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

func secretName(r *http.Request) (string, error) {
	name := strings.ToUpper(r.PathValue("name"))
	reserved := slices.ContainsFunc(reservedSecretPrefixes, func(prefix string) bool {
		return strings.HasPrefix(name, prefix)
	})
	if !secretNamePattern.MatchString(name) || reserved {
		return "", badRequest(secretNameProblem)
	}
	return name, nil
}

func (s *Server) handleListSecrets(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeAdmin)
	if err != nil {
		return err
	}
	secrets, err := s.store.Secrets(r.Context(), repo.Name)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, secrets)
}

func (s *Server) handleSetSecret(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeAdmin)
	if err != nil {
		return err
	}
	name, err := secretName(r)
	if err != nil {
		return err
	}
	body, err := decodeBody[valueRequest](w, r)
	if err != nil {
		return err
	}
	if body.Value == "" || len(body.Value) > maxSecret {
		return badRequest("a secret is 1 byte to 64 KiB")
	}
	sealed, err := s.sealSecret([]byte(body.Value))
	if err != nil {
		return err
	}
	eventID, err := s.store.SetSecret(r.Context(), identityFrom(r).actor, repo.Name, name, sealed)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, secretSet{EventID: eventID, Name: name})
}

func (s *Server) handleDeleteSecret(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeAdmin)
	if err != nil {
		return err
	}
	name, err := secretName(r)
	if err != nil {
		return err
	}
	eventID, err := s.store.DeleteSecret(r.Context(), identityFrom(r).actor, repo.Name, name)
	if err != nil {
		return missing(err, "no secret "+name)
	}
	return writeJSON(w, http.StatusOK, deletedWithEvent{Deleted: name, EventID: eventID})
}

func (s *Server) handleListVariables(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	variables, err := s.store.Variables(r.Context(), repo.Name)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, variables)
}

func (s *Server) handleSetVariable(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return err
	}
	name, err := secretName(r)
	if err != nil {
		return err
	}
	body, err := decodeBody[valueRequest](w, r)
	if err != nil {
		return err
	}
	if len(body.Value) > maxSecret {
		return badRequest("a variable is at most 64 KiB")
	}
	eventID, err := s.store.SetVariable(r.Context(), identityFrom(r).actor, repo.Name, name, body.Value)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, variableSet{EventID: eventID, Name: name, Value: body.Value})
}

func (s *Server) handleDeleteVariable(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return err
	}
	name, err := secretName(r)
	if err != nil {
		return err
	}
	eventID, err := s.store.DeleteVariable(r.Context(), identityFrom(r).actor, repo.Name, name)
	if err != nil {
		return missing(err, "no variable "+name)
	}
	return writeJSON(w, http.StatusOK, deletedWithEvent{Deleted: name, EventID: eventID})
}
