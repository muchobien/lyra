package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
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
	"sync"
	"sync/atomic"
	"time"

	"code.forgejo.org/forgejo/actions-proto/ping/v1/pingv1connect"
	"code.forgejo.org/forgejo/actions-proto/runner/v1/runnerv1connect"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry/handlers"
	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/distribution/distribution/v3/registry/storage/driver/factory"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/filesystem"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/s3-aws"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

type Config struct {
	DataDir           string
	TrustedProxies    []netip.Prefix
	AllowedLogins     []string
	SigningKeyFile    string
	Storage           string
	S3Endpoint        string
	S3Bucket          string
	S3Region          string
	RegistryHost      string
	ActionsURL        string
	DefaultActionsURL string
	SecretsKeyFile    string
	JobStaleAfter     time.Duration
	ActionsRetention  time.Duration
}

type Server struct {
	config     Config
	store      *store.Store
	git        repos.Git
	dbPath     string
	signer     ssh.Signer
	storage    driver.StorageDriver
	registry   http.Handler
	secretsKey []byte
	stop       context.CancelFunc
	gate       sync.RWMutex
	readOnly   atomic.Bool
}

const lyraUser = "lyra"

type identity struct {
	actor   store.Actor
	scope   string
	repos   []string
	tokenID int64
}

type contextKey struct{}

var scopeRank = map[string]int{"read": 1, "write": 2, "admin": 3}

var repoNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)

var reservedNames = []string{"api", "static", "healthz", "v2"}

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
	if err := db.Migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	server := &Server{
		config: config,
		store:  db,
		git:    repos.Git{},
		dbPath: dbPath,
	}
	var driverName string
	var driverParams map[string]any
	switch config.Storage {
	case "", "filesystem":
		driverName, driverParams = "filesystem", map[string]any{"rootdirectory": filepath.Join(config.DataDir, "blobs")}
	case "s3":
		driverName, driverParams = "s3aws", map[string]any{
			"regionendpoint": config.S3Endpoint, "bucket": config.S3Bucket, "region": config.S3Region, "forcepathstyle": true,
		}
	default:
		db.Close()
		return nil, fmt.Errorf("unknown storage %q: use filesystem or s3", config.Storage)
	}
	if server.storage, err = factory.Create(context.Background(), driverName, driverParams); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: %w", err)
	}
	secret := make([]byte, 32)
	rand.Read(secret)
	logrus.SetLevel(logrus.WarnLevel)
	server.registry = handlers.NewApp(context.Background(), &configuration.Configuration{
		Storage: configuration.Storage{
			driverName: driverParams,
			"delete":   configuration.Parameters{"enabled": true},
			"redirect": configuration.Parameters{"disable": true},
		},
		HTTP:    configuration.HTTP{Secret: hex.EncodeToString(secret)},
		Catalog: configuration.Catalog{MaxEntries: 1000},
	})
	if server.secretsKey, err = loadSecretsKey(config); err != nil {
		db.Close()
		return nil, err
	}
	keyFile := config.SigningKeyFile
	if keyFile == "" {
		keyFile = filepath.Join(config.DataDir, "lyra_signing_key")
		if _, err := os.Stat(keyFile); errors.Is(err, os.ErrNotExist) {
			_, privateKey, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return nil, err
			}
			block, err := ssh.MarshalPrivateKey(privateKey, "lyra merge commits")
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(keyFile, pem.EncodeToMemory(block), 0o600); err != nil {
				return nil, err
			}
			slog.Warn("generated a signing key in the data directory; it exists only there and in backups, so keep it with --signing-key-file", "file", keyFile)
		}
	}
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("signing key: %w", err)
	}
	if server.signer, err = ssh.ParsePrivateKey(keyData); err != nil {
		return nil, fmt.Errorf("signing key %s: %w", keyFile, err)
	}
	fingerprint := ssh.FingerprintSHA256(server.signer.PublicKey())
	lyraKeys, err := db.Keys(lyraUser)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(lyraKeys, func(key store.Key) bool { return key.Fingerprint == fingerprint }) {
		publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(server.signer.PublicKey()))) + " lyra"
		if _, err := db.CreateKey(store.Key{User: lyraUser, Title: "Lyra merge commits", PublicKey: publicKey, Fingerprint: fingerprint, Signing: true}); err != nil {
			return nil, fmt.Errorf("register signing key: %w", err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"pre-receive", "post-receive"} {
		hook := "#!/bin/sh\nexec '" + strings.ReplaceAll(executable, "'", `'\''`) + "' hook " + name + "\n"
		if err := os.WriteFile(filepath.Join(config.DataDir, "hooks", name), []byte(hook), 0o755); err != nil {
			return nil, err
		}
	}
	var scheduler context.Context
	scheduler, server.stop = context.WithCancel(context.Background())
	go server.schedule(scheduler)
	go server.maintain(scheduler)
	return server, nil
}

func (s *Server) Close() error {
	s.stop()
	return s.store.Close()
}

func (s *Server) repoDir(name string) string {
	return filepath.Join(s.config.DataDir, "repos", name+".git")
}

func (s *Server) Handler() http.Handler {
	router := chi.NewRouter()
	router.Use(http.NewCrossOriginProtection().Handler, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				chi.RouteContext(r.Context()).RouteMethod = http.MethodGet
			}
			next.ServeHTTP(w, r)
		})
	}, s.authenticate, s.writeGate)
	router.Handle("/v2/*", http.HandlerFunc(s.handleRegistry))
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})

	router.Route("/api", func(api chi.Router) {
		api.Get("/me", s.handleMe)
		api.Route("/repos", func(repos chi.Router) {
			repos.Get("/", s.handleListRepos)
			repos.Post("/", s.handleCreateRepo)
			repos.Route("/{owner}/{repo}", func(repo chi.Router) {
				repo.Get("/", s.handleGetRepo)
				repo.Delete("/", s.handleDeleteRepo)
				repo.Get("/commits", s.handleListCommits)
				repo.Get("/commits/{rev}", s.handleGetCommit)
				repo.Get("/tags", s.handleListTags)
				repo.Route("/issues", func(issues chi.Router) {
					issues.Get("/", s.handleListIssues)
					issues.Post("/", s.handleCreateIssue)
					issues.Get("/{number}", s.handleGetIssue)
					issues.Patch("/{number}", s.handleEditItem("issue"))
					issues.Post("/{number}/comments", s.handleCommentItem("issue"))
				})
				repo.Route("/pulls", func(pulls chi.Router) {
					pulls.Get("/", s.handleListPulls)
					pulls.Post("/", s.handleCreatePull)
					pulls.Get("/{number}", s.handleGetPull)
					pulls.Patch("/{number}", s.handleEditItem("pr"))
					pulls.Post("/{number}/comments", s.handleCommentItem("pr"))
					pulls.Get("/{number}/commits", s.handlePullCommits)
					pulls.Get("/{number}/diff", s.handlePullDiff)
					pulls.Get("/{number}/files", s.handlePullFiles)
					pulls.Post("/{number}/merge", s.handleMergePull)
					pulls.Post("/{number}/reviews", s.handleReviewPull)
				})
				repo.Get("/images", s.handleListImages)
				repo.Post("/workflows/dispatch", s.handleDispatch)
				repo.Route("/secrets", func(secrets chi.Router) {
					secrets.Get("/", s.handleListSecrets)
					secrets.Put("/{name}", s.handleSetSecret)
					secrets.Delete("/{name}", s.handleDeleteSecret)
				})
				repo.Route("/variables", func(variables chi.Router) {
					variables.Get("/", s.handleListVariables)
					variables.Put("/{name}", s.handleSetVariable)
					variables.Delete("/{name}", s.handleDeleteVariable)
				})
				repo.Route("/runs", func(runs chi.Router) {
					runs.Get("/", s.handleListRuns)
					runs.Get("/{number}", s.handleGetRun)
					runs.Get("/{number}/jobs/{job}/logs", s.handleRunLogs)
					runs.Post("/{number}/cancel", s.handleCancelRun)
					runs.Post("/{number}/rerun", s.handleRerunRun)
				})
				repo.Route("/releases", func(releases chi.Router) {
					releases.Get("/", s.handleListReleases)
					releases.Post("/", s.handleCreateRelease)
					releases.Get("/latest", s.handleLatestRelease)
					releases.Get("/notes", s.handleReleaseNotes)
					releases.Get("/tags/*", s.handleReleaseByTag)
					releases.Get("/{id}", s.handleGetRelease)
					releases.Patch("/{id}", s.handleEditRelease)
					releases.Delete("/{id}", s.handleDeleteRelease)
					releases.Post("/{id}/assets", s.handleUploadAsset)
					releases.Get("/{id}/assets/{name}", s.handleDownloadAsset)
					releases.Delete("/{id}/assets/{name}", s.handleDeleteAsset)
				})
				repo.Route("/rules", func(rules chi.Router) {
					rules.Get("/", s.handleListRules)
					rules.Post("/", s.handleSetRule)
					rules.Delete("/", s.handleDeleteRule)
				})
				repo.Route("/labels", func(labels chi.Router) {
					labels.Get("/", s.handleListLabels)
					labels.Post("/", s.handleCreateLabel)
					labels.Delete("/{label}", s.handleDeleteLabel)
				})
			})
		})
		api.Route("/tokens", func(tokens chi.Router) {
			tokens.Get("/", s.handleListTokens)
			tokens.Post("/", s.handleCreateToken)
			tokens.Delete("/{id}", s.handleDeleteToken)
		})
		api.Route("/keys", func(keys chi.Router) {
			keys.Get("/", s.handleListKeys)
			keys.Post("/", s.handleCreateKey)
			keys.Delete("/{id}", s.handleDeleteKey)
		})
		api.Get("/events", s.handleListEvents)
		api.Get("/events/stream", s.handleEventStream)
		api.Get("/signing-key", s.handleSigningKey)
		runnerPath, runnerHandler := runnerv1connect.NewRunnerServiceHandler(runnerService{server: s})
		pingPath, pingHandler := pingv1connect.NewPingServiceHandler(pingService{})
		api.Handle("/actions"+runnerPath+"*", http.StripPrefix("/api/actions", runnerHandler))
		api.Handle("/actions"+pingPath+"*", http.StripPrefix("/api/actions", pingHandler))
		api.Route("/admin", func(admin chi.Router) {
			admin.Post("/backup", s.handleBackup)
			admin.Post("/readonly", s.handleReadOnly)
			admin.Get("/status", s.handleStatus)
			admin.Get("/storage-check", s.handleStorageCheck)
			admin.Post("/registry-gc", s.handleRegistryGC)
		})
		api.Route("/owners", func(owners chi.Router) {
			owners.Get("/", s.handleListOwners)
			owners.Post("/", s.handleCreateOwner)
			owners.Get("/{owner}", s.handleGetOwner)
			owners.Delete("/{owner}", s.handleDeleteOwner)
		})
		api.Route("/runners", func(runners chi.Router) {
			runners.Get("/", s.handleListRunners)
			runners.Post("/", s.handleCreateRunner)
			runners.Delete("/{id}", s.handleDeleteRunner)
			runners.Get("/tokens", s.handleListRunnerTokens)
			runners.Post("/tokens", s.handleCreateRunnerToken)
			runners.Delete("/tokens/{id}", s.handleDeleteRunnerToken)
		})
	})

	router.Get("/", s.uiRepoList)
	router.Get("/{owner}", s.uiOwner)
	router.Route("/{owner}/{repo}", func(repo chi.Router) {
		repo.Get("/info/refs", s.handleGit)
		repo.Post("/git-upload-pack", s.handleGit)
		repo.Post("/git-receive-pack", s.handleGit)

		repo.Get("/", s.uiRepo)
		repo.Get("/tree/{ref}", s.uiTree)
		repo.Get("/tree/{ref}/*", s.uiTree)
		repo.Get("/blob/{ref}/*", s.uiBlob)
		repo.Get("/commits/{ref}", s.uiCommits)
		repo.Get("/commit/{sha}", s.uiCommit)
		repo.Get("/tags", s.uiTags)
		repo.Get("/releases", s.uiReleases)
		repo.Get("/runs", s.uiRuns)
		repo.Get("/runs/{number}", s.uiRun)
		repo.Post("/runs/{number}/{action}", s.uiRunAction)
		repo.Get("/images", s.uiImages)
		repo.Get("/releases/tag/*", s.uiReleases)
		repo.Get("/releases/download/*", s.handleStableDownload)
		repo.Get("/releases/latest/download/{name}", s.handleLatestDownload)
		repo.Post("/issues", s.uiCreateIssue)
		repo.Post("/pulls/{number}/merge", s.uiMergePull)
		repo.Post("/pulls/{number}/reviews", s.uiReviewPull)
		for kind, path := range itemPaths {
			repo.Get("/"+path, s.uiItems(kind))
			repo.Get("/"+path+"/{number}", s.uiItemPage(kind))
			repo.Post("/"+path+"/{number}/comments", s.uiCommentItem(kind))
			repo.Post("/"+path+"/{number}/edit", s.uiEditItem(kind))
		}
	})

	return router
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
		if r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/api/actions/") {
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
					id = &identity{actor: store.Actor{User: token.User, Via: "token:" + token.Name}, scope: token.Scope, repos: token.Repos, tokenID: token.ID}
				}
				if errors.Is(err, store.ErrNotFound) {
					if _, run, err := s.store.RunningTask(store.HashToken(secret)); err == nil {
						id = &identity{actor: store.Actor{User: run.Author.User, Via: fmt.Sprintf("job:%s#%d", run.Repo, run.Number)}, scope: "write", repos: []string{run.Repo}}
					}
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
	if len(id.repos) == 0 || slices.Contains(id.repos, repo) {
		return true
	}
	owner, _, _ := strings.Cut(repo, "/")
	return slices.Contains(id.repos, owner+"/*")
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
