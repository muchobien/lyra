package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

type Config struct {
	DataDir        string
	TrustedProxies []netip.Prefix
	AllowedLogins  []string
}

type Server struct {
	config Config
	store  *store.Store
	git    repos.Git
	dbPath string
}

type identity struct {
	actor store.Actor
	scope string
	repos []string
}

type contextKey struct{}

var scopeRank = map[string]int{"read": 1, "write": 2, "admin": 3}

var repoNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)

var reservedNames = []string{"api", "static", "healthz"}

func New(config Config) (*Server, error) {
	if len(config.TrustedProxies) > 0 && len(config.AllowedLogins) == 0 {
		return nil, errors.New("--allow-login is required when --trusted-proxy is set")
	}
	for _, dir := range []string{config.DataDir, filepath.Join(config.DataDir, "repos"), filepath.Join(config.DataDir, "hooks"), filepath.Join(config.DataDir, "tmp")} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, err
		}
	}
	dbPath := filepath.Join(config.DataDir, "lyra.db")
	db, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	server := &Server{
		config: config,
		store:  db,
		git:    repos.Git{AllowedSigners: filepath.Join(config.DataDir, "allowed_signers")},
		dbPath: dbPath,
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	hook := "#!/bin/sh\nexec '" + strings.ReplaceAll(executable, "'", `'\''`) + "' hook post-receive\n"
	if err := os.WriteFile(filepath.Join(config.DataDir, "hooks", "post-receive"), []byte(hook), 0o755); err != nil {
		return nil, err
	}
	if err := server.writeAllowedSigners(); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *Server) Close() error {
	return s.store.Close()
}

func (s *Server) repoDir(name string) string {
	return filepath.Join(s.config.DataDir, "repos", name+".git")
}

func (s *Server) writeAllowedSigners() error {
	keys, err := s.store.Keys("")
	if err != nil {
		return err
	}
	var lines strings.Builder
	for _, key := range keys {
		if key.Signing {
			fmt.Fprintf(&lines, "%s namespaces=\"git\" %s\n", key.User, key.PublicKey)
		}
	}
	path := s.git.AllowedSigners
	if err := os.WriteFile(path+".tmp", []byte(lines.String()), 0o640); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})

	mux.HandleFunc("GET /api/me", s.handleMe)
	mux.HandleFunc("GET /api/repos", s.handleListRepos)
	mux.HandleFunc("POST /api/repos", s.handleCreateRepo)
	mux.HandleFunc("GET /api/repos/{repo}", s.handleGetRepo)
	mux.HandleFunc("DELETE /api/repos/{repo}", s.handleDeleteRepo)
	mux.HandleFunc("GET /api/repos/{repo}/commits", s.handleListCommits)
	mux.HandleFunc("GET /api/repos/{repo}/commits/{rev}", s.handleGetCommit)
	mux.HandleFunc("GET /api/repos/{repo}/tags", s.handleListTags)
	mux.HandleFunc("GET /api/tokens", s.handleListTokens)
	mux.HandleFunc("POST /api/tokens", s.handleCreateToken)
	mux.HandleFunc("DELETE /api/tokens/{id}", s.handleDeleteToken)
	mux.HandleFunc("GET /api/keys", s.handleListKeys)
	mux.HandleFunc("POST /api/keys", s.handleCreateKey)
	mux.HandleFunc("DELETE /api/keys/{id}", s.handleDeleteKey)
	mux.HandleFunc("GET /api/events", s.handleListEvents)

	mux.HandleFunc("GET /{repo}/info/refs", s.handleGit)
	mux.HandleFunc("POST /{repo}/git-upload-pack", s.handleGit)
	mux.HandleFunc("POST /{repo}/git-receive-pack", s.handleGit)

	mux.HandleFunc("GET /{$}", s.uiRepoList)
	mux.HandleFunc("GET /{repo}", s.uiRepo)
	mux.HandleFunc("GET /{repo}/tree/{ref}/{path...}", s.uiTree)
	mux.HandleFunc("GET /{repo}/blob/{ref}/{path...}", s.uiBlob)
	mux.HandleFunc("GET /{repo}/commits/{ref}", s.uiCommits)
	mux.HandleFunc("GET /{repo}/commit/{sha}", s.uiCommit)
	mux.HandleFunc("GET /{repo}/tags", s.uiTags)

	return s.authenticate(mux)
}

func (s *Server) trustedProxy(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	for _, prefix := range s.config.TrustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		var id *identity
		login := r.Header.Get("Tailscale-User-Login")
		if login != "" && s.trustedProxy(r.RemoteAddr) {
			if !slices.Contains(s.config.AllowedLogins, login) {
				writeError(w, http.StatusForbidden, "login not allowed")
				return
			}
			id = &identity{actor: store.Actor{User: login, Via: "tailscale"}, scope: "admin"}
		}
		if id == nil {
			secret := ""
			if bearer, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); found {
				secret = strings.TrimSpace(bearer)
			} else if _, password, ok := r.BasicAuth(); ok {
				secret = password
			}
			if secret != "" {
				token, err := s.store.TokenBySecret(secret)
				if err != nil && !errors.Is(err, store.ErrNotFound) {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				if err == nil && (token.ExpiresAt == nil || token.ExpiresAt.After(time.Now())) {
					id = &identity{actor: store.Actor{User: token.User, Via: "token:" + token.Name}, scope: token.Scope, repos: token.Repos}
				}
			}
		}
		if id == nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="lyra"`)
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		id.actor.Agent = strings.TrimSpace(r.Header.Get("Lyra-Agent"))
		id.actor.Session = strings.TrimSpace(r.Header.Get("Lyra-Session"))
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "user", id.actor.User, "via", id.actor.Via, "agent", id.actor.Agent)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, id)))
	})
}

func identityFrom(r *http.Request) *identity {
	return r.Context().Value(contextKey{}).(*identity)
}

func (id *identity) can(scope, repo string) bool {
	if scopeRank[id.scope] < scopeRank[scope] {
		return false
	}
	if repo == "" {
		return len(id.repos) == 0
	}
	return len(id.repos) == 0 || slices.Contains(id.repos, repo)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
