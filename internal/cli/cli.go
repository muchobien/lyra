package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const usage = `lyra: client for the Lyra forge. Output is JSON.

Environment:
  LYRA_URL      server URL (default http://localhost:8080)
  LYRA_TOKEN    API token; not needed on the tailnet behind tailscale-auth
  LYRA_AGENT    agent name recorded on writes (Lyra-Agent header)
  LYRA_SESSION  session id recorded on writes (Lyra-Session header)

Commands:
  lyra serve [--listen :8080] [--data DIR] [--trusted-proxy IPS] [--allow-login LOGINS]
  lyra me
  lyra repo create <name> [--description TEXT]
  lyra repo list
  lyra repo get <name>
  lyra repo delete <name> --yes
  lyra commit list <repo> [--ref REF] [--limit N]
  lyra commit show <repo> <rev>
  lyra tag list <repo>
  lyra token create --name NAME --scope read|write|admin [--repo NAME]... [--expires 720h]
  lyra token list
  lyra token revoke <id>
  lyra key add --signing [--agent NAME] [--title TEXT] <public-key-file|->
  lyra key list
  lyra key delete <id>
  lyra event list [--repo NAME] [--limit N]
`

type listFlag []string

func (values *listFlag) String() string {
	return strings.Join(*values, ",")
}

func (values *listFlag) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type usageError string

func (err usageError) Error() string {
	return string(err)
}

func parse(flags *flag.FlagSet, args []string) ([]string, error) {
	flags.SetOutput(io.Discard)
	positionals := []string{}
	for {
		if err := flags.Parse(args); err != nil {
			return nil, usageError(err.Error())
		}
		args = flags.Args()
		if len(args) == 0 {
			return positionals, nil
		}
		positionals = append(positionals, args[0])
		args = args[1:]
	}
}

func Run(args []string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(usage)
		return 0
	}
	output, err := run(args)
	if err != nil {
		encoded, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintln(os.Stderr, string(encoded))
		if _, isUsage := err.(usageError); isUsage {
			return 2
		}
		return 1
	}
	os.Stdout.Write(output)
	return 0
}

func command(args []string, count int) ([]string, error) {
	if len(args) != count {
		return nil, usageError("wrong number of arguments; run lyra help")
	}
	return args, nil
}

func run(args []string) ([]byte, error) {
	group := args[0]
	action := ""
	rest := args[1:]
	if group != "me" {
		if len(args) < 2 {
			return nil, usageError("missing subcommand; run lyra help")
		}
		action = args[1]
		rest = args[2:]
	}
	flags := flag.NewFlagSet(group+" "+action, flag.ContinueOnError)

	switch group + " " + action {
	case "me ":
		if _, err := command(rest, 0); err != nil {
			return nil, err
		}
		return request(http.MethodGet, "/api/me", nil)

	case "repo create":
		description := flags.String("description", "", "")
		positionals, err := parse(flags, rest)
		if err == nil {
			_, err = command(positionals, 1)
		}
		if err != nil {
			return nil, err
		}
		return request(http.MethodPost, "/api/repos", map[string]string{"name": positionals[0], "description": *description})

	case "repo list":
		if _, err := command(rest, 0); err != nil {
			return nil, err
		}
		return request(http.MethodGet, "/api/repos", nil)

	case "repo get":
		positionals, err := command(rest, 1)
		if err != nil {
			return nil, err
		}
		return request(http.MethodGet, "/api/repos/"+url.PathEscape(positionals[0]), nil)

	case "repo delete":
		yes := flags.Bool("yes", false, "")
		positionals, err := parse(flags, rest)
		if err == nil {
			_, err = command(positionals, 1)
		}
		if err != nil {
			return nil, err
		}
		if !*yes {
			return nil, usageError("repo delete removes the repository permanently; pass --yes")
		}
		return request(http.MethodDelete, "/api/repos/"+url.PathEscape(positionals[0]), nil)

	case "commit list":
		ref := flags.String("ref", "", "")
		limit := flags.Int("limit", 30, "")
		positionals, err := parse(flags, rest)
		if err == nil {
			_, err = command(positionals, 1)
		}
		if err != nil {
			return nil, err
		}
		query := url.Values{"limit": {fmt.Sprint(*limit)}}
		if *ref != "" {
			query.Set("ref", *ref)
		}
		return request(http.MethodGet, "/api/repos/"+url.PathEscape(positionals[0])+"/commits?"+query.Encode(), nil)

	case "commit show":
		positionals, err := command(rest, 2)
		if err != nil {
			return nil, err
		}
		return request(http.MethodGet, "/api/repos/"+url.PathEscape(positionals[0])+"/commits/"+url.PathEscape(positionals[1]), nil)

	case "tag list":
		positionals, err := command(rest, 1)
		if err != nil {
			return nil, err
		}
		return request(http.MethodGet, "/api/repos/"+url.PathEscape(positionals[0])+"/tags", nil)

	case "token create":
		name := flags.String("name", "", "")
		scope := flags.String("scope", "", "")
		expires := flags.String("expires", "", "")
		var repos listFlag
		flags.Var(&repos, "repo", "")
		positionals, err := parse(flags, rest)
		if err == nil {
			_, err = command(positionals, 0)
		}
		if err != nil {
			return nil, err
		}
		if *name == "" || *scope == "" {
			return nil, usageError("token create needs --name and --scope")
		}
		return request(http.MethodPost, "/api/tokens", map[string]any{"name": *name, "scope": *scope, "repos": []string(repos), "expires_in": *expires})

	case "token list":
		if _, err := command(rest, 0); err != nil {
			return nil, err
		}
		return request(http.MethodGet, "/api/tokens", nil)

	case "token revoke":
		positionals, err := command(rest, 1)
		if err != nil {
			return nil, err
		}
		return request(http.MethodDelete, "/api/tokens/"+url.PathEscape(positionals[0]), nil)

	case "key add":
		signing := flags.Bool("signing", false, "")
		agent := flags.String("agent", "", "")
		title := flags.String("title", "", "")
		positionals, err := parse(flags, rest)
		if err == nil {
			_, err = command(positionals, 1)
		}
		if err != nil {
			return nil, err
		}
		if !*signing {
			return nil, usageError("only signing keys are supported; pass --signing")
		}
		var publicKey []byte
		if positionals[0] == "-" {
			publicKey, err = io.ReadAll(os.Stdin)
		} else {
			publicKey, err = os.ReadFile(positionals[0])
		}
		if err != nil {
			return nil, err
		}
		if bytes.Contains(publicKey, []byte("PRIVATE KEY")) {
			return nil, usageError("that is a private key; pass the .pub file")
		}
		return request(http.MethodPost, "/api/keys", map[string]any{"signing": true, "agent": *agent, "title": *title, "public_key": strings.TrimSpace(string(publicKey))})

	case "key list":
		if _, err := command(rest, 0); err != nil {
			return nil, err
		}
		return request(http.MethodGet, "/api/keys", nil)

	case "key delete":
		positionals, err := command(rest, 1)
		if err != nil {
			return nil, err
		}
		return request(http.MethodDelete, "/api/keys/"+url.PathEscape(positionals[0]), nil)

	case "event list":
		repo := flags.String("repo", "", "")
		limit := flags.Int("limit", 50, "")
		positionals, err := parse(flags, rest)
		if err == nil {
			_, err = command(positionals, 0)
		}
		if err != nil {
			return nil, err
		}
		query := url.Values{"limit": {fmt.Sprint(*limit)}}
		if *repo != "" {
			query.Set("repo", *repo)
		}
		return request(http.MethodGet, "/api/events?"+query.Encode(), nil)
	}
	return nil, usageError("unknown command " + strings.TrimSpace(group+" "+action) + "; run lyra help")
}

func request(method, path string, body any) ([]byte, error) {
	base := strings.TrimRight(os.Getenv("LYRA_URL"), "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := os.Getenv("LYRA_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if agent := os.Getenv("LYRA_AGENT"); agent != "" {
		req.Header.Set("Lyra-Agent", agent)
	}
	if session := os.Getenv("LYRA_SESSION"); session != "" {
		req.Header.Set("Lyra-Session", session)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		var apiError struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &apiError) == nil && apiError.Error != "" {
			return nil, fmt.Errorf("%s (HTTP %d)", apiError.Error, response.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}
