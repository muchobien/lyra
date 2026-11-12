package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/muchobien/lyra/internal/cli"
	"github.com/muchobien/lyra/internal/server"
	"github.com/muchobien/lyra/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		if err := serve(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "lyra serve:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "hook" && os.Args[2] == "post-receive" {
		if err := postReceive(); err != nil {
			fmt.Fprintln(os.Stderr, "lyra hook:", err)
			os.Exit(1)
		}
		return
	}
	os.Exit(cli.Run(os.Args[1:]))
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func serve(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	listen := flags.String("listen", envOr("LYRA_LISTEN", ":8080"), "listen address (LYRA_LISTEN)")
	dataDir := flags.String("data", envOr("LYRA_DATA", "data"), "data directory (LYRA_DATA)")
	proxies := flags.String("trusted-proxy", os.Getenv("LYRA_TRUSTED_PROXY"), "comma-separated IPs or CIDRs allowed to set Tailscale-User-Login (LYRA_TRUSTED_PROXY)")
	logins := flags.String("allow-login", os.Getenv("LYRA_ALLOW_LOGIN"), "comma-separated Tailscale logins allowed in (LYRA_ALLOW_LOGIN)")
	flags.Parse(args)

	config := server.Config{}
	absolute, err := filepath.Abs(*dataDir)
	if err != nil {
		return err
	}
	config.DataDir = absolute
	for _, value := range strings.Split(*proxies, ",") {
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
	for _, value := range strings.Split(*logins, ",") {
		if value = strings.TrimSpace(value); value != "" {
			config.AllowedLogins = append(config.AllowedLogins, value)
		}
	}
	instance, err := server.New(config)
	if err != nil {
		return err
	}
	defer instance.Close()
	slog.Info("listening", "addr", *listen, "data", config.DataDir, "trusted_proxies", config.TrustedProxies)
	httpServer := &http.Server{Addr: *listen, Handler: instance.Handler(), ReadHeaderTimeout: 10 * time.Second}
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
	return db.AddEvent("push", os.Getenv("LYRA_REPO"), actor, map[string]any{"refs": updates})
}
