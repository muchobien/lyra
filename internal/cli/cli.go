package cli

import (
	"bufio"
	"bytes"
	"context"
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

var ErrTimeout = errors.New("timeout")

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

	emit := func(cmd *cobra.Command, output []byte) error {
		if jq == "" {
			_, err := cmd.OutOrStdout().Write(output)
			return err
		}
		return filter(cmd.OutOrStdout(), output, jq)
	}
	call := func(cmd *cobra.Command, method, path string, body any) error {
		output, err := request(method, path, body)
		if err != nil {
			return err
		}
		return emit(cmd, output)
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
	var waitRepo string
	var waitActions []string
	var since int64
	var timeout time.Duration
	eventWait := &cobra.Command{
		Use:   "wait",
		Short: "Wait for the next matching event and print it",
		Long: `Wait for the next event that matches the filters and print it as JSON.

Without --since, only events after the command starts match. Pass the id of an
event you have already seen as --since to also match anything that happened
since then. The command reconnects if the connection drops, without missing
events. It exits 3 with {"error":"timeout"} when --timeout passes.`,
		Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if waitRepo != "" {
				query.Set("repo", waitRepo)
			}
			for _, action := range waitActions {
				query.Add("action", action)
			}
			if cmd.Flags().Changed("since") {
				if since < 0 {
					return UsageError{errors.New("--since must be an event id")}
				}
				query.Set("since", fmt.Sprint(since))
			}
			output, err := waitForEvent("/api/events/stream?"+query.Encode(), timeout)
			if err != nil {
				return err
			}
			var indented bytes.Buffer
			if err := json.Indent(&indented, output, "", "  "); err != nil {
				return err
			}
			indented.WriteByte('\n')
			return emit(cmd, indented.Bytes())
		},
	}
	eventWait.Flags().StringVar(&waitRepo, "repo", "", "only events for this repository")
	eventWait.Flags().StringArrayVar(&waitActions, "action", nil, "only these actions, as glob patterns such as push or repo.* (repeatable)")
	eventWait.Flags().Int64Var(&since, "since", 0, "match events after this event id (default: from now)")
	eventWait.Flags().DurationVar(&timeout, "timeout", 0, "give up after this long, such as 30m (default: wait forever)")
	event.AddCommand(eventList, eventWait)

	itemCommands := func(use, segment, noun, states string) *cobra.Command {
		parent := &cobra.Command{Use: use, Short: "Manage " + noun + "s"}
		itemsPath := func(repo string) string {
			return "/api/repos/" + url.PathEscape(repo) + "/" + segment
		}
		var itemTitle, head, base string
		var itemLabels, itemAssignees []string
		createCommand := &cobra.Command{
			Use:   "create <repo>",
			Short: "Open a new " + noun,
			Args:  exactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if itemTitle == "" {
					return UsageError{errors.New(use + " create needs --title")}
				}
				body, _, err := readBody(cmd)
				if err != nil {
					return err
				}
				create := map[string]any{"title": itemTitle, "body": body, "labels": itemLabels, "assignees": itemAssignees}
				if segment == "pulls" {
					if head == "" {
						return UsageError{errors.New("pr create needs --head")}
					}
					create["head"], create["base"] = head, base
				}
				return call(cmd, http.MethodPost, itemsPath(args[0]), create)
			},
		}
		createCommand.Flags().StringVar(&itemTitle, "title", "", "title (required)")
		if segment == "pulls" {
			createCommand.Flags().StringVar(&head, "head", "", "branch with the changes (required)")
			createCommand.Flags().StringVar(&base, "base", "", "branch to merge into (default: the default branch)")
		}
		addBodyFlags(createCommand)
		createCommand.Flags().StringArrayVar(&itemLabels, "label", nil, "add this label, which must exist (repeatable)")
		createCommand.Flags().StringArrayVar(&itemAssignees, "assignee", nil, "assign a login or agent name, or @me (repeatable)")

		var listState string
		var listLabels, listAssignees []string
		var listLimit int
		listCommand := &cobra.Command{
			Use:   "list <repo>",
			Short: "List " + noun + "s, newest first",
			Args:  exactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				query := url.Values{"state": {listState}, "limit": {fmt.Sprint(listLimit)}, "label": listLabels, "assignee": listAssignees}
				return call(cmd, http.MethodGet, itemsPath(args[0])+"?"+query.Encode(), nil)
			},
		}
		listCommand.Flags().StringVar(&listState, "state", "open", states)
		listCommand.Flags().StringArrayVar(&listLabels, "label", nil, "only "+noun+"s with this label (repeatable)")
		listCommand.Flags().StringArrayVar(&listAssignees, "assignee", nil, "only "+noun+"s assigned to this login or agent, or @me (repeatable)")
		listCommand.Flags().IntVar(&listLimit, "limit", 30, "maximum number of "+noun+"s")

		viewCommand := &cobra.Command{
			Use:   "view <repo> <number>",
			Short: "Show a " + noun + " with its comments",
			Args:  exactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return call(cmd, http.MethodGet, itemsPath(args[0])+"/"+url.PathEscape(args[1]), nil)
			},
		}

		commentCommand := &cobra.Command{
			Use:   "comment <repo> <number>",
			Short: "Comment on a " + noun,
			Args:  exactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				body, set, err := readBody(cmd)
				if err != nil {
					return err
				}
				if !set {
					return UsageError{errors.New(use + " comment needs --body or --body-file")}
				}
				return call(cmd, http.MethodPost, itemsPath(args[0])+"/"+url.PathEscape(args[1])+"/comments", map[string]string{"body": body})
			},
		}
		addBodyFlags(commentCommand)

		var editTitle string
		var addLabels, removeLabels, addAssignees, removeAssignees []string
		editCommand := &cobra.Command{
			Use:   "edit <repo> <number>",
			Short: "Change a " + noun + "'s title, body, labels or assignees",
			Args:  exactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				edit := map[string]any{}
				if cmd.Flags().Changed("title") {
					edit["title"] = editTitle
				}
				body, set, err := readBody(cmd)
				if err != nil {
					return err
				}
				if set {
					edit["body"] = body
				}
				for key, values := range map[string][]string{"add_labels": addLabels, "remove_labels": removeLabels, "add_assignees": addAssignees, "remove_assignees": removeAssignees} {
					if len(values) > 0 {
						edit[key] = values
					}
				}
				if len(edit) == 0 {
					return UsageError{errors.New(use + " edit needs at least one change; see lyra " + use + " edit --help")}
				}
				return call(cmd, http.MethodPatch, itemsPath(args[0])+"/"+url.PathEscape(args[1]), edit)
			},
		}
		editCommand.Flags().StringVar(&editTitle, "title", "", "new title")
		addBodyFlags(editCommand)
		editCommand.Flags().StringArrayVar(&addLabels, "add-label", nil, "add a label (repeatable)")
		editCommand.Flags().StringArrayVar(&removeLabels, "remove-label", nil, "remove a label (repeatable)")
		editCommand.Flags().StringArrayVar(&addAssignees, "add-assignee", nil, "assign a login or agent name, or @me (repeatable)")
		editCommand.Flags().StringArrayVar(&removeAssignees, "remove-assignee", nil, "unassign a login or agent name, or @me (repeatable)")

		for _, change := range []struct{ use, short, state string }{
			{"close <repo> <number>", "Close a " + noun, "closed"},
			{"reopen <repo> <number>", "Reopen a closed " + noun, "open"},
		} {
			parent.AddCommand(&cobra.Command{
				Use:   change.use,
				Short: change.short,
				Args:  exactArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					return call(cmd, http.MethodPatch, itemsPath(args[0])+"/"+url.PathEscape(args[1]), map[string]string{"state": change.state})
				},
			})
		}
		parent.AddCommand(createCommand, listCommand, viewCommand, commentCommand, editCommand)
		return parent
	}
	issue := itemCommands("issue", "issues", "issue", "open, closed or all")
	pr := itemCommands("pr", "pulls", "pull request", "open, closed, merged or all")
	pullPath := func(args []string) string {
		return "/api/repos/" + url.PathEscape(args[0]) + "/pulls/" + url.PathEscape(args[1])
	}
	prCommits := &cobra.Command{
		Use:   "commits <repo> <number>",
		Short: "List a pull request's commits with signature status",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, pullPath(args)+"/commits", nil)
		},
	}
	var filesOnly bool
	prDiff := &cobra.Command{
		Use:   "diff <repo> <number>",
		Short: "Print a pull request's diff, or its changed files with --files",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filesOnly {
				return call(cmd, http.MethodGet, pullPath(args)+"/files", nil)
			}
			return call(cmd, http.MethodGet, pullPath(args)+"/diff", nil)
		},
	}
	prDiff.Flags().BoolVar(&filesOnly, "files", false, "list changed files as JSON instead")
	pr.AddCommand(prCommits, prDiff)

	label := &cobra.Command{Use: "label", Short: "Manage a repository's labels"}
	labelsPath := func(repo string) string {
		return "/api/repos/" + url.PathEscape(repo) + "/labels"
	}
	var color, labelDescription string
	labelCreate := &cobra.Command{
		Use:   "create <repo> <name>",
		Short: "Create a label",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPost, labelsPath(args[0]), map[string]string{"name": args[1], "color": color, "description": labelDescription})
		},
	}
	labelCreate.Flags().StringVar(&color, "color", "", "six hex digits, such as d73a4a")
	labelCreate.Flags().StringVar(&labelDescription, "description", "", "label description")
	labelList := &cobra.Command{
		Use:   "list <repo>",
		Short: "List labels",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, labelsPath(args[0]), nil)
		},
	}
	labelDelete := &cobra.Command{
		Use:   "delete <repo> <name>",
		Short: "Delete a label and remove it from every issue",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, labelsPath(args[0])+"/"+url.PathEscape(args[1]), nil)
		},
	}
	label.AddCommand(labelCreate, labelList, labelDelete)

	root.AddCommand(me, repo, commit, tag, token, key, event, issue, pr, label)
	return root
}

func addBodyFlags(cmd *cobra.Command) {
	cmd.Flags().String("body", "", "body text in Markdown")
	cmd.Flags().String("body-file", "", "read the body from this file, or - for stdin")
}

func readBody(cmd *cobra.Command) (string, bool, error) {
	flags := cmd.Flags()
	if flags.Changed("body") && flags.Changed("body-file") {
		return "", false, UsageError{errors.New("pass --body or --body-file, not both")}
	}
	if flags.Changed("body-file") {
		file, _ := flags.GetString("body-file")
		var data []byte
		var err error
		if file == "-" {
			data, err = io.ReadAll(cmd.InOrStdin())
		} else {
			data, err = os.ReadFile(file)
		}
		return string(data), true, err
	}
	body, _ := flags.GetString("body")
	return body, flags.Changed("body"), nil
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

func newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
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
	req, err := http.NewRequestWithContext(ctx, method, base+path, reader)
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
	return req, nil
}

func responseError(response *http.Response) error {
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var apiError struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(data, &apiError) == nil && apiError.Error != "" {
		return fmt.Errorf("%s (HTTP %d)", apiError.Error, response.StatusCode)
	}
	return fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
}

func request(method, path string, body any) ([]byte, error) {
	req, err := newRequest(context.Background(), method, path, body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, responseError(response)
	}
	return io.ReadAll(response.Body)
}

func waitForEvent(path string, timeout time.Duration) ([]byte, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	lastID := ""
	for {
		req, err := newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "text/event-stream")
		if lastID != "" {
			req.Header.Set("Last-Event-ID", lastID)
		}
		response, err := http.DefaultClient.Do(req)
		if err == nil {
			if response.StatusCode >= 400 && response.StatusCode < 500 {
				defer response.Body.Close()
				return nil, responseError(response)
			}
			reader := bufio.NewReader(response.Body)
			data := []byte{}
			for response.StatusCode == http.StatusOK {
				line, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				line = strings.TrimRight(line, "\r\n")
				if value, found := strings.CutPrefix(line, "id:"); found {
					lastID = strings.TrimSpace(value)
				} else if value, found := strings.CutPrefix(line, "data:"); found {
					data = append(data, strings.TrimPrefix(value, " ")...)
				} else if line == "" && len(data) > 0 {
					response.Body.Close()
					return data, nil
				}
			}
			response.Body.Close()
		}
		select {
		case <-ctx.Done():
			return nil, ErrTimeout
		case <-time.After(time.Second):
		}
	}
}
