package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/muchobien/lyra/internal/store"
)

const (
	scopeRead  = "read"
	scopeWrite = "write"
	scopeAdmin = "admin"
)

var scopeRank = map[string]int{scopeRead: 1, scopeWrite: 2, scopeAdmin: 3}

type identity struct {
	actor   store.Actor
	scope   string
	repos   []string
	tokenID int64
}

type contextKey struct{}

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
	if len(id.repos) == 0 || slices.Contains(id.repos, repo) {
		return true
	}
	owner, _, _ := strings.Cut(repo, "/")
	return slices.Contains(id.repos, owner+"/*")
}

func (id *identity) name() string {
	if id.actor.Agent != "" {
		return id.actor.Agent
	}
	return id.actor.User
}

func (id *identity) resolve(names []string) []string {
	resolved := []string{}
	for _, name := range names {
		if name == "@me" {
			name = id.name()
		}
		resolved = append(resolved, name)
	}
	return resolved
}

func (id *identity) seesOwner(owner string) bool {
	return len(id.repos) == 0 || slices.ContainsFunc(id.repos, func(repo string) bool {
		return strings.HasPrefix(repo, owner+"/")
	})
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
	return slices.ContainsFunc(s.config.TrustedProxies, func(prefix netip.Prefix) bool {
		return prefix.Contains(addr)
	})
}

func requestSecret(r *http.Request) string {
	if bearer, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); found {
		return strings.TrimSpace(bearer)
	}
	if _, password, ok := r.BasicAuth(); ok {
		return password
	}
	return ""
}

func (s *Server) secretIdentity(ctx context.Context, secret string) (*identity, error) {
	token, err := s.store.TokenBySecret(ctx, secret)
	if errors.Is(err, store.ErrNotFound) {
		_, run, err := s.store.RunningTask(ctx, store.HashToken(secret))
		if err != nil {
			return nil, nil
		}
		return &identity{
			actor: store.Actor{User: run.Author.User, Via: fmt.Sprintf("job:%s#%d", run.Repo, run.Number)},
			scope: scopeWrite,
			repos: []string{run.Repo},
		}, nil
	}
	if err != nil {
		return nil, err
	}
	if token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now()) {
		return nil, nil
	}
	return &identity{
		actor:   store.Actor{User: token.User, Via: "token:" + token.Name},
		scope:   token.Scope,
		repos:   token.Repos,
		tokenID: token.ID,
	}, nil
}

func (s *Server) identify(r *http.Request) (*identity, error) {
	login := r.Header.Get("Tailscale-User-Login")
	if login != "" && s.trustedProxy(r.RemoteAddr) {
		if !slices.Contains(s.config.AllowedLogins, login) {
			return nil, forbidden("login not allowed")
		}
		return &identity{actor: store.Actor{User: login, Via: "tailscale"}, scope: scopeAdmin}, nil
	}
	secret := requestSecret(r)
	if secret == "" {
		return nil, nil
	}
	return s.secretIdentity(r.Context(), secret)
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/api/actions/") {
			next.ServeHTTP(w, r)
			return
		}
		id, err := s.identify(r)
		if err != nil {
			status, message := errorStatus(err)
			writeError(w, status, message)
			return
		}
		if id == nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="lyra"`)
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		id.actor.Agent = strings.TrimSpace(r.Header.Get("Lyra-Agent"))
		id.actor.Session = strings.TrimSpace(r.Header.Get("Lyra-Session"))
		slog.Info(
			"request",
			"method", r.Method,
			"path", r.URL.Path,
			"user", id.actor.User,
			"via", id.actor.Via,
			"agent", id.actor.Agent,
		)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, id)))
	})
}
