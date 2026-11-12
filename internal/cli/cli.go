package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

type UsageError struct {
	err error
}

func (usage UsageError) Error() string {
	return usage.err.Error()
}

func exactArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != count {
			return UsageError{fmt.Errorf("%s takes %d argument(s), got %d; see %s --help", cmd.CommandPath(), count, len(args), cmd.CommandPath())}
		}
		return nil
	}
}

func Root() *cobra.Command {
	var jq string
	root := &cobra.Command{
		Use:   "lyra",
		Short: "Client for the Lyra forge",
		Long: `Client for the Lyra forge. Output is JSON on stdout; errors are {"error": "..."} on stderr.

Environment:
  LYRA_URL      server URL (default http://localhost:8080)
  LYRA_TOKEN    API token; not needed on the tailnet behind tailscale-auth
  LYRA_AGENT    agent name recorded on writes (Lyra-Agent header)
  LYRA_SESSION  session id recorded on writes (Lyra-Session header)`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().StringVar(&jq, "jq", "", "filter JSON output with a jq expression")
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return UsageError{err}
	})

	call := func(cmd *cobra.Command, method, path string, body any) error {
		output, err := request(method, path, body)
		if err != nil {
			return err
		}
		if jq == "" {
			_, err = cmd.OutOrStdout().Write(output)
			return err
		}
		return filter(cmd.OutOrStdout(), output, jq)
	}

	me := &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated actor and scope",
		Args:  exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/me", nil)
		},
	}

	repo := &cobra.Command{Use: "repo", Short: "Manage repositories"}
	var description string
	repoCreate := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a repository",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPost, "/api/repos", map[string]string{"name": args[0], "description": description})
		},
	}
	repoCreate.Flags().StringVar(&description, "description", "", "repository description")
	repoList := &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		Args:  exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/repos", nil)
		},
	}
	repoGet := &cobra.Command{
		Use:   "get <name>",
		Short: "Show a repository with its branches and clone URL",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/repos/"+url.PathEscape(args[0]), nil)
		},
	}
	var yes bool
	repoDelete := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a repository permanently",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return UsageError{errors.New("repo delete removes the repository permanently; pass --yes")}
			}
			return call(cmd, http.MethodDelete, "/api/repos/"+url.PathEscape(args[0]), nil)
		},
	}
	repoDelete.Flags().BoolVar(&yes, "yes", false, "confirm permanent deletion")
	repo.AddCommand(repoCreate, repoList, repoGet, repoDelete)

	commit := &cobra.Command{Use: "commit", Short: "Read commits and their signature status"}
	var ref string
	var commitLimit int
	commitList := &cobra.Command{
		Use:   "list <repo>",
		Short: "List commits with signature status",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{"limit": {fmt.Sprint(commitLimit)}}
			if ref != "" {
				query.Set("ref", ref)
			}
			return call(cmd, http.MethodGet, "/api/repos/"+url.PathEscape(args[0])+"/commits?"+query.Encode(), nil)
		},
	}
	commitList.Flags().StringVar(&ref, "ref", "", "branch, tag or commit (default: the default branch)")
	commitList.Flags().IntVar(&commitLimit, "limit", 30, "maximum number of commits")
	commitShow := &cobra.Command{
		Use:   "show <repo> <rev>",
		Short: "Show one commit, verified with git verify-commit",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/repos/"+url.PathEscape(args[0])+"/commits/"+url.PathEscape(args[1]), nil)
		},
	}
	commit.AddCommand(commitList, commitShow)

	tag := &cobra.Command{Use: "tag", Short: "Read tags and their signature status"}
	tag.AddCommand(&cobra.Command{
		Use:   "list <repo>",
		Short: "List tags, verified with git verify-tag",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/repos/"+url.PathEscape(args[0])+"/tags", nil)
		},
	})

	token := &cobra.Command{Use: "token", Short: "Manage API tokens"}
	var tokenName, scope, expires string
	var tokenRepos []string
	tokenCreate := &cobra.Command{
		Use:   "create",
		Short: "Create a token; the secret is shown once",
		Args:  exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tokenName == "" || scope == "" {
				return UsageError{errors.New("token create needs --name and --scope")}
			}
			return call(cmd, http.MethodPost, "/api/tokens", map[string]any{"name": tokenName, "scope": scope, "repos": tokenRepos, "expires_in": expires})
		},
	}
	tokenCreate.Flags().StringVar(&tokenName, "name", "", "token name, recorded as via token:<name>")
	tokenCreate.Flags().StringVar(&scope, "scope", "", "read, write or admin")
	tokenCreate.Flags().StringArrayVar(&tokenRepos, "repo", nil, "limit the token to this repository (repeatable)")
	tokenCreate.Flags().StringVar(&expires, "expires", "", "lifetime as a Go duration, such as 720h")
	tokenList := &cobra.Command{
		Use:   "list",
		Short: "List tokens",
		Args:  exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/tokens", nil)
		},
	}
	tokenRevoke := &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke a token",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, "/api/tokens/"+url.PathEscape(args[0]), nil)
		},
	}
	token.AddCommand(tokenCreate, tokenList, tokenRevoke)

	key := &cobra.Command{Use: "key", Short: "Manage SSH signing keys"}
	var signing bool
	var agent, title string
	keyAdd := &cobra.Command{
		Use:   "add <public-key-file|->",
		Short: "Register an SSH public key for commit and tag signatures",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !signing {
				return UsageError{errors.New("only signing keys are supported; pass --signing")}
			}
			var publicKey []byte
			var err error
			if args[0] == "-" {
				publicKey, err = io.ReadAll(cmd.InOrStdin())
			} else {
				publicKey, err = os.ReadFile(args[0])
			}
			if err != nil {
				return err
			}
			if bytes.Contains(publicKey, []byte("PRIVATE KEY")) {
				return UsageError{errors.New("that is a private key; pass the .pub file")}
			}
			return call(cmd, http.MethodPost, "/api/keys", map[string]any{"signing": true, "agent": agent, "title": title, "public_key": strings.TrimSpace(string(publicKey))})
		},
	}
	keyAdd.Flags().BoolVar(&signing, "signing", false, "register as a signing key (required)")
	keyAdd.Flags().StringVar(&agent, "agent", "", "label the key as belonging to this agent")
	keyAdd.Flags().StringVar(&title, "title", "", "key title (default: the key comment)")
	keyList := &cobra.Command{
		Use:   "list",
		Short: "List keys",
		Args:  exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/keys", nil)
		},
	}
	keyDelete := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a key",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, "/api/keys/"+url.PathEscape(args[0]), nil)
		},
	}
	key.AddCommand(keyAdd, keyList, keyDelete)

	event := &cobra.Command{Use: "event", Short: "Read the event log"}
	var eventRepo string
	var eventLimit int
	eventList := &cobra.Command{
		Use:   "list",
		Short: "List recent events with their actors",
		Args:  exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{"limit": {fmt.Sprint(eventLimit)}}
			if eventRepo != "" {
				query.Set("repo", eventRepo)
			}
			return call(cmd, http.MethodGet, "/api/events?"+query.Encode(), nil)
		},
	}
	eventList.Flags().StringVar(&eventRepo, "repo", "", "only events for this repository")
	eventList.Flags().IntVar(&eventLimit, "limit", 50, "maximum number of events")
	event.AddCommand(eventList)

	root.AddCommand(me, repo, commit, tag, token, key, event)
	return root
}

func filter(out io.Writer, output []byte, expression string) error {
	query, err := gojq.Parse(expression)
	if err != nil {
		return UsageError{fmt.Errorf("--jq: %w", err)}
	}
	var input any
	if err := json.Unmarshal(output, &input); err != nil {
		return err
	}
	results := query.Run(input)
	for {
		value, ok := results.Next()
		if !ok {
			return nil
		}
		if err, isError := value.(error); isError {
			return fmt.Errorf("--jq: %w", err)
		}
		if text, isString := value.(string); isString {
			fmt.Fprintln(out, text)
			continue
		}
		encoded, err := gojq.Marshal(value)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(encoded))
	}
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
