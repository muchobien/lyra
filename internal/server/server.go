package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry/handlers"
	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/distribution/distribution/v3/registry/storage/driver/factory"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/filesystem"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/s3-aws"
	"github.com/prometheus/client_golang/prometheus"
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
	metrics    *prometheus.Registry
	requests   *prometheus.CounterVec
	durations  *prometheus.HistogramVec
}

const lyraUser = "lyra"

var hookNames = []string{"pre-receive", "post-receive"}

func New(ctx context.Context, config Config) (*Server, error) {
	if len(config.TrustedProxies) > 0 && len(config.AllowedLogins) == 0 {
		return nil, errors.New("--allow-login is required when --trusted-proxy is set")
	}
	for _, name := range []string{"", "repos", "hooks", "tmp"} {
		if err := os.MkdirAll(filepath.Join(config.DataDir, name), 0o750); err != nil {
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
		git:    repos.Git{},
		dbPath: dbPath,
	}
	if err := server.start(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return server, nil
}

func (s *Server) start(ctx context.Context) error {
	if err := s.store.Migrate(ctx); err != nil {
		return err
	}
	if err := s.openStorage(ctx); err != nil {
		return err
	}
	var err error
	if s.secretsKey, err = loadSecretsKey(s.config); err != nil {
		return err
	}
	if err := s.loadSigner(ctx); err != nil {
		return err
	}
	if err := s.installHooks(); err != nil {
		return err
	}
	s.registerMetrics()
	var background context.Context
	background, s.stop = context.WithCancel(ctx)
	go s.schedule(background)
	go s.maintain(background)
	return nil
}

func (s *Server) openStorage(ctx context.Context) error {
	var driverName string
	var driverParams map[string]any
	switch s.config.Storage {
	case "", "filesystem":
		driverName = "filesystem"
		driverParams = map[string]any{"rootdirectory": filepath.Join(s.config.DataDir, "blobs")}
	case "s3":
		driverName = "s3aws"
		driverParams = map[string]any{
			"regionendpoint": s.config.S3Endpoint,
			"bucket":         s.config.S3Bucket,
			"region":         s.config.S3Region,
			"forcepathstyle": true,
		}
	default:
		return fmt.Errorf("unknown storage %q: use filesystem or s3", s.config.Storage)
	}
	var err error
	if s.storage, err = factory.Create(ctx, driverName, driverParams); err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	secret := make([]byte, 32)
	rand.Read(secret)
	logrus.SetLevel(logrus.WarnLevel)
	s.registry = handlers.NewApp(context.WithoutCancel(ctx), &configuration.Configuration{
		Storage: configuration.Storage{
			driverName: driverParams,
			"delete":   configuration.Parameters{"enabled": true},
			"redirect": configuration.Parameters{"disable": true},
		},
		HTTP:    configuration.HTTP{Secret: hex.EncodeToString(secret)},
		Catalog: configuration.Catalog{MaxEntries: 1000},
	})
	return nil
}

func generateSigningKey(keyFile string) error {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	block, err := ssh.MarshalPrivateKey(privateKey, "lyra merge commits")
	if err != nil {
		return err
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(block), 0o600); err != nil {
		return err
	}
	slog.Warn(
		"generated a signing key in the data directory; it exists only there and in backups, "+
			"so keep it with --signing-key-file",
		"file", keyFile,
	)
	return nil
}

func (s *Server) loadSigner(ctx context.Context) error {
	keyFile := s.config.SigningKeyFile
	if keyFile == "" {
		keyFile = filepath.Join(s.config.DataDir, "lyra_signing_key")
		if _, err := os.Stat(keyFile); errors.Is(err, os.ErrNotExist) {
			if err := generateSigningKey(keyFile); err != nil {
				return err
			}
		}
	}
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("signing key: %w", err)
	}
	if s.signer, err = ssh.ParsePrivateKey(keyData); err != nil {
		return fmt.Errorf("signing key %s: %w", keyFile, err)
	}
	fingerprint := ssh.FingerprintSHA256(s.signer.PublicKey())
	lyraKeys, err := s.store.Keys(ctx, lyraUser)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(lyraKeys, func(key store.Key) bool { return key.Fingerprint == fingerprint }) {
		return nil
	}
	_, err = s.store.CreateKey(ctx, store.Key{
		User:        lyraUser,
		Title:       "Lyra merge commits",
		PublicKey:   strings.TrimSpace(string(ssh.MarshalAuthorizedKey(s.signer.PublicKey()))) + " lyra",
		Fingerprint: fingerprint,
		Signing:     true,
	})
	if err != nil {
		return fmt.Errorf("register signing key: %w", err)
	}
	return nil
}

func (s *Server) installHooks() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	for _, name := range hookNames {
		hook := "#!/bin/sh\nexec '" + strings.ReplaceAll(executable, "'", `'\''`) + "' hook " + name + "\n"
		if err := os.WriteFile(filepath.Join(s.config.DataDir, "hooks", name), []byte(hook), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) Close() error {
	s.stop()
	return s.store.Close()
}

func (s *Server) repoDir(name string) string {
	return filepath.Join(s.config.DataDir, "repos", name+".git")
}
