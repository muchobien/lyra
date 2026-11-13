package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/backup"
	"github.com/muchobien/lyra/internal/cli"
	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/server"
	"github.com/muchobien/lyra/internal/store"
)

func main() {
	root := cli.Root()

	var listen, dataDir, signingKey string
	var storage, s3Endpoint, s3Bucket, s3Region, registryHost, actionsURL, defaultActionsURL, secretsKey string
	var jobStaleAfter, actionsRetention time.Duration
	var metricsListen string
	var proxies, logins []string
	serveCommand := &cobra.Command{
		Use:   "serve",
		Short: "Run the Lyra server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return serve(listen, metricsListen, proxies, logins, server.Config{
				DataDir: dataDir, SigningKeyFile: signingKey, Storage: storage, S3Endpoint: s3Endpoint, S3Bucket: s3Bucket, S3Region: s3Region, RegistryHost: registryHost, ActionsURL: actionsURL, DefaultActionsURL: defaultActionsURL, SecretsKeyFile: secretsKey, JobStaleAfter: jobStaleAfter, ActionsRetention: actionsRetention,
			})
		},
	}
	serveCommand.Flags().StringVar(&signingKey, "signing-key-file", os.Getenv("LYRA_SIGNING_KEY_FILE"), "SSH private key Lyra signs merge commits with (LYRA_SIGNING_KEY_FILE; default: generated in the data directory)")
	serveCommand.Flags().StringVar(&listen, "listen", envOr("LYRA_LISTEN", ":8080"), "listen address (LYRA_LISTEN)")
	serveCommand.Flags().StringVar(&metricsListen, "metrics-listen", os.Getenv("LYRA_METRICS_LISTEN"), "address to serve Prometheus metrics on, such as :9090, apart from the main listener (LYRA_METRICS_LISTEN; default: off)")
	serveCommand.Flags().StringVar(&dataDir, "data", envOr("LYRA_DATA", "data"), "data directory (LYRA_DATA)")
	serveCommand.Flags().StringSliceVar(&proxies, "trusted-proxy", splitEnv("LYRA_TRUSTED_PROXY"), "IPs or CIDRs allowed to set Tailscale-User-Login (LYRA_TRUSTED_PROXY, comma-separated)")
	serveCommand.Flags().StringSliceVar(&logins, "allow-login", splitEnv("LYRA_ALLOW_LOGIN"), "Tailscale logins allowed in (LYRA_ALLOW_LOGIN, comma-separated)")
	serveCommand.Flags().StringVar(&storage, "storage", envOr("LYRA_STORAGE", "filesystem"), "where release assets and registry blobs are stored: filesystem (in the data directory) or s3 (LYRA_STORAGE)")
	serveCommand.Flags().StringVar(&s3Endpoint, "s3-endpoint", os.Getenv("LYRA_S3_ENDPOINT"), "S3 endpoint URL, such as http://garage:3900 (LYRA_S3_ENDPOINT); credentials come from AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
	serveCommand.Flags().StringVar(&s3Bucket, "s3-bucket", os.Getenv("LYRA_S3_BUCKET"), "S3 bucket (LYRA_S3_BUCKET)")
	serveCommand.Flags().StringVar(&s3Region, "s3-region", envOr("LYRA_S3_REGION", "garage"), "S3 region (LYRA_S3_REGION)")
	serveCommand.Flags().StringVar(&registryHost, "registry-host", os.Getenv("LYRA_REGISTRY_HOST"), "host name images are pulled from, such as registry.example.com (LYRA_REGISTRY_HOST; default: the request's host)")
	serveCommand.Flags().StringVar(&actionsURL, "actions-url", os.Getenv("LYRA_ACTIONS_URL"), "URL runners and jobs reach Lyra at, such as http://lyra.lyra.svc.cluster.local:8080 (LYRA_ACTIONS_URL; default: the request's URL)")
	serveCommand.Flags().StringVar(&defaultActionsURL, "default-actions-url", envOr("LYRA_DEFAULT_ACTIONS_URL", "https://data.forgejo.org"), "where short action names like actions/checkout@v4 are fetched from (LYRA_DEFAULT_ACTIONS_URL)")
	serveCommand.Flags().StringVar(&secretsKey, "secrets-key-file", os.Getenv("LYRA_SECRETS_KEY_FILE"), "file with 64 hex characters that encrypt Actions secrets (LYRA_SECRETS_KEY_FILE; default: generated in the data directory)")
	serveCommand.Flags().DurationVar(&jobStaleAfter, "job-stale-after", envDuration("LYRA_JOB_STALE_AFTER", 10*time.Minute), "fail a running job when its runner hasn't reported for this long (LYRA_JOB_STALE_AFTER)")
	serveCommand.Flags().DurationVar(&actionsRetention, "actions-retention", envDuration("LYRA_ACTIONS_RETENTION", 90*24*time.Hour), "delete finished runs and their logs after this long; 0 keeps them (LYRA_ACTIONS_RETENTION)")

	hookCommand := &cobra.Command{Use: "hook", Short: "Git hooks run by the server", Hidden: true}
	for _, name := range []string{"pre-receive", "post-receive"} {
		hookCommand.AddCommand(&cobra.Command{
			Use:  name,
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runHook(name)
			},
		})
	}
	var restoreData string
	restoreCommand := &cobra.Command{
		Use:   "restore <backup.tar|->",
		Short: "Restore a backup from lyra admin backup into a new data directory",
		Long: `Restore a backup from lyra admin backup into a new, empty data directory, then check it:
the database's integrity, a schema no newer than this binary, git fsck of every repository,
and every repository's refs against the manifest. Prints a JSON report. Run lyra serve
on the directory afterwards, with the same signing and secrets keys and the same storage.
Then lyra admin storage-check tells whether storage has every file the database refers to.`,
		Args: cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if restoreData == "" {
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
			report, err := backup.Restore(input, restoreData)
			if err != nil {
				return err
			}
			encoded, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(encoded))
			return nil
		},
	}
	restoreCommand.Flags().StringVar(&restoreData, "data", "", "new, empty data directory to restore into (required)")
	root.AddCommand(serveCommand, hookCommand, restoreCommand)

	if err := root.Execute(); err != nil {
		encoded, _ := json.Marshal(map[string]string{"error": err.Error()})
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

func serve(listen, metricsListen string, proxies, logins []string, config server.Config) error {
	absolute, err := filepath.Abs(config.DataDir)
	if err != nil {
		return err
	}
	config.DataDir = absolute
	for _, value := range proxies {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !strings.Contains(value, "/") {
			addr, err := netip.ParseAddr(value)
			if err != nil {
				return fmt.Errorf("trusted proxy %q: %w", value, err)
			}
			value = netip.PrefixFrom(addr, addr.BitLen()).String()
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return fmt.Errorf("trusted proxy %q: %w", value, err)
		}
		config.TrustedProxies = append(config.TrustedProxies, prefix.Masked())
	}
	for _, value := range logins {
		if value = strings.TrimSpace(value); value != "" {
			config.AllowedLogins = append(config.AllowedLogins, value)
		}
	}
	instance, err := server.New(config)
	if err != nil {
		return err
	}
	defer instance.Close()
	slog.Info("listening", "addr", listen, "data", config.DataDir, "trusted_proxies", config.TrustedProxies, "storage", config.Storage)
	if metricsListen != "" {
		metrics := &http.Server{Addr: metricsListen, Handler: instance.MetricsHandler(), ReadHeaderTimeout: 10 * time.Second}
		go func() {
			if err := metrics.ListenAndServe(); err != nil {
				slog.Error("metrics listener", "error", err)
			}
		}()
		slog.Info("serving metrics", "addr", metricsListen)
	}
	httpServer := &http.Server{Addr: listen, Handler: instance.Handler(), ReadHeaderTimeout: 10 * time.Second}
	return httpServer.ListenAndServe()
}

func runHook(name string) error {
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
		if err := server.CheckPush(db, repos.Git{}, ".", repo, updates); err != nil {
			fmt.Fprintln(os.Stderr, "lyra: "+err.Error())
			db.Close()
			os.Exit(1)
		}
		return nil
	}
	actor := store.Actor{User: os.Getenv("LYRA_USER"), Via: os.Getenv("LYRA_VIA"), Agent: os.Getenv("LYRA_AGENT"), Session: os.Getenv("LYRA_SESSION")}
	if _, err := db.AddEvent("push", repo, actor, map[string]any{"refs": updates}); err != nil {
		return err
	}
	messages, err := server.ProcessPush(db, repos.Git{}, ".", repo, actor, updates)
	for _, message := range messages {
		fmt.Println("lyra: " + message)
	}
	return err
}
