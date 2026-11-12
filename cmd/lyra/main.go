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

	"github.com/muchobien/lyra/internal/cli"
	"github.com/muchobien/lyra/internal/server"
	"github.com/muchobien/lyra/internal/store"
)

func main() {
	root := cli.Root()

	var listen, dataDir string
	var proxies, logins []string
	serveCommand := &cobra.Command{
		Use:   "serve",
		Short: "Run the Lyra server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return serve(listen, dataDir, proxies, logins)
		},
	}
	serveCommand.Flags().StringVar(&listen, "listen", envOr("LYRA_LISTEN", ":8080"), "listen address (LYRA_LISTEN)")
	serveCommand.Flags().StringVar(&dataDir, "data", envOr("LYRA_DATA", "data"), "data directory (LYRA_DATA)")
	serveCommand.Flags().StringSliceVar(&proxies, "trusted-proxy", splitEnv("LYRA_TRUSTED_PROXY"), "IPs or CIDRs allowed to set Tailscale-User-Login (LYRA_TRUSTED_PROXY, comma-separated)")
	serveCommand.Flags().StringSliceVar(&logins, "allow-login", splitEnv("LYRA_ALLOW_LOGIN"), "Tailscale logins allowed in (LYRA_ALLOW_LOGIN, comma-separated)")

	hookCommand := &cobra.Command{Use: "hook", Short: "Git hooks run by the server", Hidden: true}
	hookCommand.AddCommand(&cobra.Command{
		Use:  "post-receive",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return postReceive()
		},
	})
	root.AddCommand(serveCommand, hookCommand)

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

func splitEnv(name string) []string {
	if value := os.Getenv(name); value != "" {
		return strings.Split(value, ",")
	}
	return nil
}

func serve(listen, dataDir string, proxies, logins []string) error {
	config := server.Config{}
	absolute, err := filepath.Abs(dataDir)
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
	slog.Info("listening", "addr", listen, "data", config.DataDir, "trusted_proxies", config.TrustedProxies)
	httpServer := &http.Server{Addr: listen, Handler: instance.Handler(), ReadHeaderTimeout: 10 * time.Second}
	return httpServer.ListenAndServe()
}

func postReceive() error {
	db, err := store.Open(os.Getenv("LYRA_DB"))
	if err != nil {
		return err
	}
	defer db.Close()
	type update struct {
		Ref string `json:"ref"`
		Old string `json:"old"`
		New string `json:"new"`
	}
	updates := []update{}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 3 {
			updates = append(updates, update{Old: fields[0], New: fields[1], Ref: fields[2]})
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	actor := store.Actor{User: os.Getenv("LYRA_USER"), Via: os.Getenv("LYRA_VIA"), Agent: os.Getenv("LYRA_AGENT"), Session: os.Getenv("LYRA_SESSION")}
	_, err = db.AddEvent("push", os.Getenv("LYRA_REPO"), actor, map[string]any{"refs": updates})
	return err
}
