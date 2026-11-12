package server

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hiddeco/sshsig"
	"golang.org/x/crypto/ssh"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

type repoView struct {
	store.Repo
	DefaultBranch string      `json:"default_branch"`
	Branches      []repos.Ref `json:"branches"`
	CloneURL      string      `json:"clone_url"`
}

func cloneURL(r *http.Request, name string) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/" + name + ".git"
}

func (s *Server) repoView(r *http.Request, repo store.Repo) (repoView, error) {
	dir := s.repoDir(repo.Name)
	branches, err := s.git.Branches(dir)
	if err != nil {
		return repoView{}, err
	}
	return repoView{Repo: repo, DefaultBranch: s.git.DefaultBranch(dir), Branches: branches, CloneURL: cloneURL(r, repo.Name)}, nil
}

func decodeBody(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

func (s *Server) lookupRepo(w http.ResponseWriter, r *http.Request, scope string) (store.Repo, bool) {
	name := r.PathValue("repo")
	if !identityFrom(r).can(scope, name) {
		writeError(w, http.StatusForbidden, "token lacks "+scope+" access to "+name)
		return store.Repo{}, false
	}
	repo, err := s.store.Repo(name)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "repository not found")
		return repo, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return repo, false
	}
	return repo, true
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	writeJSON(w, http.StatusOK, map[string]any{"actor": id.actor, "scope": id.scope, "repos": id.repos})
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	all, err := s.store.Repos()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	views := []repoView{}
	for _, repo := range all {
		if !id.can("read", repo.Name) {
			continue
		}
		view, err := s.repoView(r, repo)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		views = append(views, view)
	}
	writeJSON(w, http.StatusOK, views)
}

func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if !repoNamePattern.MatchString(body.Name) || strings.HasSuffix(body.Name, ".git") || slices.Contains(reservedNames, body.Name) {
		writeError(w, http.StatusBadRequest, "invalid repository name")
		return
	}
	if !id.can("admin", body.Name) {
		writeError(w, http.StatusForbidden, "token lacks admin access to "+body.Name)
		return
	}
	repo, err := s.store.CreateRepo(body.Name, body.Description, id.actor.User)
	if store.IsUniqueViolation(err) {
		writeError(w, http.StatusConflict, "repository already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.git.Init(s.repoDir(repo.Name)); err != nil {
		s.store.DeleteRepo(repo.Name)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.store.AddEvent("repo.create", repo.Name, id.actor, map[string]string{"description": repo.Description}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	view, err := s.repoView(r, repo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *Server) handleGetRepo(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	view, err := s.repoView(r, repo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "admin")
	if !ok {
		return
	}
	if err := s.store.DeleteRepo(repo.Name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.RemoveAll(s.repoDir(repo.Name)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.store.AddEvent("repo.delete", repo.Name, identityFrom(r).actor, map[string]string{}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": repo.Name})
}

func (s *Server) verifySignatures(signatures ...*repos.Signature) error {
	keys, err := s.store.Keys("")
	if err != nil {
		return err
	}
	return verifyWithKeys(keys, signatures...)
}

func verifyWithKeys(keys []store.Key, signatures ...*repos.Signature) error {
	for _, signature := range signatures {
		if signature.Armored == nil {
			continue
		}
		parsed, err := sshsig.Unarmor(signature.Armored)
		if err != nil {
			continue
		}
		signature.Fingerprint = ssh.FingerprintSHA256(parsed.PublicKey)
		for _, key := range keys {
			if !key.Signing || key.Fingerprint != signature.Fingerprint {
				continue
			}
			publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.PublicKey))
			if err != nil {
				return err
			}
			if sshsig.Verify(bytes.NewReader(signature.Payload), parsed, publicKey, parsed.HashAlgorithm, "git") == nil {
				signature.Status = repos.Verified
				signature.KeyID = key.ID
				signature.KeyUser = key.User
				signature.KeyAgent = key.Agent
			}
		}
	}
	return nil
}

func (s *Server) commits(repo, ref string, limit int, exclude ...string) ([]repos.Commit, error) {
	commits, err := s.git.Log(s.repoDir(repo), ref, limit, exclude...)
	if err != nil {
		return nil, err
	}
	signatures := []*repos.Signature{}
	for index := range commits {
		signatures = append(signatures, &commits[index].Signature)
	}
	return commits, s.verifySignatures(signatures...)
}

func (s *Server) handleListCommits(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = s.git.DefaultBranch(s.repoDir(repo.Name))
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 || limit > 500 {
		limit = 30
	}
	commits, err := s.commits(repo.Name, ref, limit)
	if errors.Is(err, repos.ErrNotFound) {
		writeError(w, http.StatusNotFound, "unknown ref "+ref)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, commits)
}

func (s *Server) handleGetCommit(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	commit, err := s.git.Commit(s.repoDir(repo.Name), r.PathValue("rev"))
	if errors.Is(err, repos.ErrNotFound) {
		writeError(w, http.StatusNotFound, "unknown revision")
		return
	}
	if err == nil {
		err = s.verifySignatures(&commit.Signature)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, commit)
}

func (s *Server) tags(repo string) ([]repos.Tag, error) {
	tags, err := s.git.Tags(s.repoDir(repo))
	if err != nil {
		return nil, err
	}
	signatures := []*repos.Signature{}
	for index := range tags {
		signatures = append(signatures, &tags[index].Signature)
	}
	return tags, s.verifySignatures(signatures...)
}

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.lookupRepo(w, r, "read")
	if !ok {
		return
	}
	tags, err := s.tags(repo.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("admin", "") {
		writeError(w, http.StatusForbidden, "token management needs an unrestricted admin token")
		return
	}
	tokens, err := s.store.Tokens(id.actor.User)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("admin", "") {
		writeError(w, http.StatusForbidden, "token management needs an unrestricted admin token")
		return
	}
	var body struct {
		Name      string   `json:"name"`
		Scope     string   `json:"scope"`
		Repos     []string `json:"repos"`
		ExpiresIn string   `json:"expires_in"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if _, ok := scopeRank[body.Scope]; !ok {
		writeError(w, http.StatusBadRequest, "scope must be read, write or admin")
		return
	}
	if body.Repos == nil {
		body.Repos = []string{}
	}
	var expiresAt *time.Time
	if body.ExpiresIn != "" {
		duration, err := time.ParseDuration(body.ExpiresIn)
		if err != nil || duration <= 0 {
			writeError(w, http.StatusBadRequest, "expires_in must be a positive Go duration such as 720h")
			return
		}
		at := time.Now().Add(duration)
		expiresAt = &at
	}
	random := make([]byte, 32)
	rand.Read(random)
	secret := "lyra_" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(random))
	token, err := s.store.CreateToken(body.Name, id.actor.User, secret, body.Scope, body.Repos, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.store.AddEvent("token.create", "", id.actor, map[string]any{"id": token.ID, "name": token.Name, "scope": token.Scope, "repos": token.Repos}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "secret": secret})
}

func (s *Server) handleDeleteToken(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("admin", "") {
		writeError(w, http.StatusForbidden, "token management needs an unrestricted admin token")
		return
	}
	tokenID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid token id")
		return
	}
	if err := s.store.DeleteToken(id.actor.User, tokenID); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "token not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.store.AddEvent("token.revoke", "", id.actor, map[string]any{"id": tokenID}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": tokenID})
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("read", "") {
		writeError(w, http.StatusForbidden, "listing keys needs an unrestricted token")
		return
	}
	keys, err := s.store.Keys(id.actor.User)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("admin", "") {
		writeError(w, http.StatusForbidden, "key management needs an unrestricted admin token")
		return
	}
	var body struct {
		Title     string `json:"title"`
		Agent     string `json:"agent"`
		PublicKey string `json:"public_key"`
		Signing   bool   `json:"signing"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if !body.Signing {
		writeError(w, http.StatusBadRequest, "only signing keys are supported; set signing to true")
		return
	}
	parsed, comment, options, rest, err := ssh.ParseAuthorizedKey([]byte(body.PublicKey))
	if err != nil || len(options) > 0 || strings.TrimSpace(string(rest)) != "" {
		writeError(w, http.StatusBadRequest, "public_key must be one OpenSSH public key line")
		return
	}
	title := body.Title
	if title == "" {
		title = comment
	}
	publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(parsed)))
	key, err := s.store.CreateKey(store.Key{User: id.actor.User, Title: title, Agent: body.Agent, PublicKey: publicKey, Fingerprint: ssh.FingerprintSHA256(parsed), Signing: true})
	if store.IsUniqueViolation(err) {
		writeError(w, http.StatusConflict, "key already registered")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.store.AddEvent("key.add", "", id.actor, map[string]any{"id": key.ID, "fingerprint": key.Fingerprint, "agent": key.Agent}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, key)
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	if !id.can("admin", "") {
		writeError(w, http.StatusForbidden, "key management needs an unrestricted admin token")
		return
	}
	keyID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid key id")
		return
	}
	if err := s.store.DeleteKey(id.actor.User, keyID); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "key not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := s.store.AddEvent("key.delete", "", id.actor, map[string]any{"id": keyID}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": keyID})
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	repo := r.URL.Query().Get("repo")
	if !id.can("read", repo) {
		writeError(w, http.StatusForbidden, "token lacks read access")
		return
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 50
	}
	events, err := s.store.Events(repo, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}
