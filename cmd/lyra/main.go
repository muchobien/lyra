package main

import (
	"bufio"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/backup"
	"github.com/muchobien/lyra/internal/cli"
	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/server"
	"github.com/muchobien/lyra/internal/store"
)

type serveOptions struct {
	listen        string
	metricsListen string
	proxies       []string
	logins        []string
	config        server.Config
}

type failure struct {
	Error string `json:"error"`
}

var escaped = json.JoinOptions(
	jsontext.EscapeForHTML(true),
	jsontext.EscapeForJS(true),
	jsontext.AllowInvalidUTF8(true),
)

var (
	version = ""
	commit  = ""
	date    = ""
)

type buildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
}

func currentBuild() buildInfo {
	build := buildInfo{Version: version, Commit: commit, Date: date, Go: runtime.Version()}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return build
	}
	if build.Version == "" {
		build.Version = strings.TrimPrefix(info.Main.Version, "v")
	}
	for _, setting := range info.Settings {
		switch {
		case setting.Key == "vcs.revision" && build.Commit == "":
			build.Commit = setting.Value
		case setting.Key == "vcs.time" && build.Date == "":
			build.Date = setting.Value
		}
	}
	return build
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the version, commit and build date of this binary",
		Args:  cli.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			encoded, err := json.Marshal(currentBuild(), escaped, jsontext.WithIndent("  "))
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
			return err
		},
	}
}

func main() {
	root := cli.Root()
	root.AddCommand(serveCommand(), hookCommand(), restoreCommand(), versionCommand())

	if err := root.Execute(); err != nil {
		encoded, _ := json.Marshal(failure{Error: err.Error()}, escaped)
		fmt.Fprintln(os.Stderr, string(encoded))
		var usage cli.UsageError
		if errors.As(err, &usage) {
			os.Exit(2)
		}
		if errors.Is(err, cli.ErrTimeout) {
			os.Exit(3)
		}
		os.Exit(1)
	}
}

func serveCommand() *cobra.Command {
	var options serveOptions
	command := &cobra.Command{
		Use:   "serve",
		Short: "Run the Lyra server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return serve(cmd.Context(), options)
		},
	}
	addListenFlags(command, &options)
	addStorageFlags(command, &options.config)
	addActionsFlags(command, &options.config)
	return command
}

func addListenFlags(command *cobra.Command, options *serveOptions) {
	flags := command.Flags()
	flags.StringVar(
		&options.config.SigningKeyFile,
		"signing-key-file",
		os.Getenv("LYRA_SIGNING_KEY_FILE"),
		"SSH private key Lyra signs merge commits with "+
			"(LYRA_SIGNING_KEY_FILE; default: generated in the data directory)",
	)
	flags.StringVar(&options.listen, "listen", envOr("LYRA_LISTEN", ":8080"), "listen address (LYRA_LISTEN)")
	flags.StringVar(
		&options.metricsListen,
		"metrics-listen",
		os.Getenv("LYRA_METRICS_LISTEN"),
		"address to serve Prometheus metrics on, such as :9090, apart from the main listener "+
			"(LYRA_METRICS_LISTEN; default: off)",
	)
	flags.StringVar(&options.config.DataDir, "data", envOr("LYRA_DATA", "data"), "data directory (LYRA_DATA)")
	flags.StringSliceVar(
		&options.proxies,
		"trusted-proxy",
		splitEnv("LYRA_TRUSTED_PROXY"),
		"IPs or CIDRs allowed to set Tailscale-User-Login (LYRA_TRUSTED_PROXY, comma-separated)",
	)
	flags.StringSliceVar(
		&options.logins,
		"allow-login",
		splitEnv("LYRA_ALLOW_LOGIN"),
		"Tailscale logins allowed in (LYRA_ALLOW_LOGIN, comma-separated)",
	)
}

func addStorageFlags(command *cobra.Command, config *server.Config) {
	flags := command.Flags()
	flags.StringVar(
		&config.Storage,
		"storage",
		envOr("LYRA_STORAGE", "filesystem"),
		"where release assets and registry blobs are stored: "+
			"filesystem (in the data directory) or s3 (LYRA_STORAGE)",
	)
	flags.StringVar(
		&config.S3Endpoint,
		"s3-endpoint",
		os.Getenv("LYRA_S3_ENDPOINT"),
		"S3 endpoint URL, such as http://garage:3900 (LYRA_S3_ENDPOINT); "+
			"credentials come from AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY",
	)
	flags.StringVar(&config.S3Bucket, "s3-bucket", os.Getenv("LYRA_S3_BUCKET"), "S3 bucket (LYRA_S3_BUCKET)")
	flags.StringVar(
		&config.S3Region,
		"s3-region",
		envOr("LYRA_S3_REGION", "garage"),
		"S3 region (LYRA_S3_REGION)",
	)
	flags.StringVar(
		&config.RegistryHost,
		"registry-host",
		os.Getenv("LYRA_REGISTRY_HOST"),
		"host name images are pulled from, such as registry.example.com "+
			"(LYRA_REGISTRY_HOST; default: the request's host)",
	)
}

func addActionsFlags(command *cobra.Command, config *server.Config) {
	flags := command.Flags()
	flags.StringVar(
		&config.ActionsURL,
		"actions-url",
		os.Getenv("LYRA_ACTIONS_URL"),
		"URL runners and jobs reach Lyra at, such as http://lyra.lyra.svc.cluster.local:8080 "+
			"(LYRA_ACTIONS_URL; default: the request's URL)",
	)
	flags.StringVar(
		&config.DefaultActionsURL,
		"default-actions-url",
		envOr("LYRA_DEFAULT_ACTIONS_URL", "https://data.forgejo.org"),
		"where short action names like actions/checkout@v4 are fetched from (LYRA_DEFAULT_ACTIONS_URL)",
	)
	flags.StringSliceVar(
		&config.ActionsOwners,
		"actions-owners",
		splitEnv("LYRA_ACTIONS_OWNERS"),
		"owners whose repositories hold actions: cloned without authentication, and they run no workflows "+
			"(LYRA_ACTIONS_OWNERS, comma-separated)",
	)
	flags.StringVar(
		&config.SecretsKeyFile,
		"secrets-key-file",
		os.Getenv("LYRA_SECRETS_KEY_FILE"),
		"file with 64 hex characters that encrypt Actions secrets "+
			"(LYRA_SECRETS_KEY_FILE; default: generated in the data directory)",
	)
	flags.DurationVar(
		&config.JobStaleAfter,
		"job-stale-after",
		envDuration("LYRA_JOB_STALE_AFTER", 10*time.Minute),
		"fail a running job when its runner hasn't reported for this long (LYRA_JOB_STALE_AFTER)",
	)
	flags.DurationVar(
		&config.ActionsRetention,
		"actions-retention",
		envDuration("LYRA_ACTIONS_RETENTION", 90*24*time.Hour),
		"delete finished runs and their logs after this long; 0 keeps them (LYRA_ACTIONS_RETENTION)",
	)
}

func hookCommand() *cobra.Command {
	command := &cobra.Command{Use: "hook", Short: "Git hooks run by the server", Hidden: true}
	for _, name := range []string{"pre-receive", "post-receive"} {
		command.AddCommand(&cobra.Command{
			Use:  name,
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runHook(cmd.Context(), name)
			},
		})
	}
	return command
}

func restoreCommand() *cobra.Command {
	var dataDir string
	command := &cobra.Command{
		Use:   "restore <backup.tar|->",
		Short: "Restore a backup from lyra admin backup into a new data directory",
		Long: `Restore a backup from lyra admin backup into a new, empty data directory, then check it:
the database's integrity, a schema no newer than this binary, git fsck of every repository,
and every repository's refs against the manifest. Prints a JSON report. Run lyra serve
on the directory afterwards, with the same signing and secrets keys and the same storage.
Then lyra admin storage-check tells whether storage has every file the database refers to.`,
		Args: cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dataDir == "" {
				return cli.UsageError{Err: errors.New("--data is required")}
			}
			input := cmd.InOrStdin()
			if args[0] != "-" {
				file, err := os.Open(args[0])
				if err != nil {
					return err
				}
				defer file.Close()
				input = file
			}
			report, err := backup.Restore(cmd.Context(), input, dataDir)
			if err != nil {
				return err
			}
			encoded, err := json.Marshal(report, escaped, jsontext.WithIndent("  "))
			if err != nil {
				return err
			}
			fmt.Println(string(encoded))
			return nil
		},
	}
	command.Flags().StringVar(&dataDir, "data", "", "new, empty data directory to restore into (required)")
	return command
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envDuration(name string, fallback time.Duration) time.Duration {
	if value, err := time.ParseDuration(os.Getenv(name)); err == nil {
		return value
	}
	return fallback
}

func splitEnv(name string) []string {
	if value := os.Getenv(name); value != "" {
		return strings.Split(value, ",")
	}
	return nil
}

func trustedProxies(values []string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !strings.Contains(value, "/") {
			addr, err := netip.ParseAddr(value)
			if err != nil {
				return nil, fmt.Errorf("trusted proxy %q: %w", value, err)
			}
			value = netip.PrefixFrom(addr, addr.BitLen()).String()
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy %q: %w", value, err)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

func serve(ctx context.Context, options serveOptions) error {
	config := options.config
	absolute, err := filepath.Abs(config.DataDir)
	if err != nil {
		return err
	}
	config.DataDir = absolute
	if config.TrustedProxies, err = trustedProxies(options.proxies); err != nil {
		return err
	}
	for _, value := range options.logins {
		if value = strings.TrimSpace(value); value != "" {
			config.AllowedLogins = append(config.AllowedLogins, value)
		}
	}
	instance, err := server.New(ctx, config)
	if err != nil {
		return err
	}
	defer instance.Close()
	slog.Info(
		"listening",
		"addr", options.listen,
		"data", config.DataDir,
		"trusted_proxies", config.TrustedProxies,
		"storage", config.Storage,
	)
	if options.metricsListen != "" {
		metrics := &http.Server{
			Addr:              options.metricsListen,
			Handler:           instance.MetricsHandler(),
			ReadHeaderTimeout: 10 * time.Second,
		}
		go func() {
			if err := metrics.ListenAndServe(); err != nil {
				slog.Error("metrics listener", "error", err)
			}
		}()
		slog.Info("serving metrics", "addr", options.metricsListen)
	}
	httpServer := &http.Server{
		Addr:              options.listen,
		Handler:           instance.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return httpServer.ListenAndServe()
}

func runHook(ctx context.Context, name string) error {
	db, err := store.Open(os.Getenv("LYRA_DB"))
	if err != nil {
		return err
	}
	defer db.Close()
	updates := []server.RefUpdate{}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 3 {
			updates = append(updates, server.RefUpdate{Old: fields[0], New: fields[1], Ref: fields[2]})
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	repo := os.Getenv("LYRA_REPO")
	if name == "pre-receive" {
		if err := server.CheckPush(ctx, db, repos.Git{}, ".", repo, updates); err != nil {
			fmt.Fprintln(os.Stderr, "lyra: "+err.Error())
			db.Close()
			os.Exit(1)
		}
		return nil
	}
	actor := store.Actor{
		User:    os.Getenv("LYRA_USER"),
		Via:     os.Getenv("LYRA_VIA"),
		Agent:   os.Getenv("LYRA_AGENT"),
		Session: os.Getenv("LYRA_SESSION"),
	}
	if _, err := db.AddEvent(ctx, "push", repo, actor, server.PushEvent{Refs: updates}); err != nil {
		return err
	}
	queueRuns := os.Getenv("LYRA_ACTIONS_REPO") != "true"
	messages, err := server.ProcessPush(ctx, db, repos.Git{}, ".", repo, actor, updates, queueRuns)
	for _, message := range messages {
		fmt.Println("lyra: " + message)
	}
	return err
}
