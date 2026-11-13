package cli

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

var ErrTimeout = errors.New("timeout")

type UsageError struct {
	Err error
}

func (usage UsageError) Error() string {
	return usage.Err.Error()
}

func ExactArgs(count int) cobra.PositionalArgs {
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
  LYRA_SESSION  session id recorded on writes (Lyra-Session header)

Repositories are named <owner>/<name>, such as krosf/lyra; create the owner first with
lyra owner create.`,
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
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/me", nil)
		},
	}

	repo := &cobra.Command{Use: "repo", Short: "Manage repositories"}
	var description string
	repoCreate := &cobra.Command{
		Use:   "create <owner>/<name>",
		Short: "Create a repository under an existing owner",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPost, "/api/repos", map[string]string{"name": args[0], "description": description})
		},
	}
	repoCreate.Flags().StringVar(&description, "description", "", "repository description")
	repoList := &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/repos", nil)
		},
	}
	repoGet := &cobra.Command{
		Use:   "get <name>",
		Short: "Show a repository with its branches and clone URL",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, repoAPI(args[0]), nil)
		},
	}
	var yes bool
	repoDelete := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a repository permanently",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return UsageError{errors.New("repo delete removes the repository permanently; pass --yes")}
			}
			return call(cmd, http.MethodDelete, repoAPI(args[0]), nil)
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
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{"limit": {fmt.Sprint(commitLimit)}}
			if ref != "" {
				query.Set("ref", ref)
			}
			return call(cmd, http.MethodGet, repoAPI(args[0])+"/commits?"+query.Encode(), nil)
		},
	}
	commitList.Flags().StringVar(&ref, "ref", "", "branch, tag or commit (default: the default branch)")
	commitList.Flags().IntVar(&commitLimit, "limit", 30, "maximum number of commits")
	commitShow := &cobra.Command{
		Use:   "show <repo> <rev>",
		Short: "Show one commit, verified with git verify-commit",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, repoAPI(args[0])+"/commits/"+url.PathEscape(args[1]), nil)
		},
	}
	commit.AddCommand(commitList, commitShow)

	tag := &cobra.Command{Use: "tag", Short: "Read tags and their signature status"}
	tag.AddCommand(&cobra.Command{
		Use:   "list <repo>",
		Short: "List tags, verified with git verify-tag",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, repoAPI(args[0])+"/tags", nil)
		},
	})

	token := &cobra.Command{Use: "token", Short: "Manage API tokens"}
	var tokenName, scope, expires string
	var tokenRepos []string
	tokenCreate := &cobra.Command{
		Use:   "create",
		Short: "Create a token; the secret is shown once",
		Args:  ExactArgs(0),
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
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/tokens", nil)
		},
	}
	tokenRevoke := &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke a token",
		Args:  ExactArgs(1),
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
		Args:  ExactArgs(1),
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
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/keys", nil)
		},
	}
	keyDelete := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a key",
		Args:  ExactArgs(1),
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
		Args:  ExactArgs(0),
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
	waitAndEmit := func(cmd *cobra.Command, query url.Values, timeout time.Duration) error {
		flags := cmd.Flags()
		if flags.Changed("since") {
			since, _ := flags.GetInt64("since")
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
	}
	var waitRepo, waitAssignee string
	var waitActions []string
	var timeout time.Duration
	eventWait := &cobra.Command{
		Use:   "wait",
		Short: "Wait for the next matching event and print it",
		Long: `Wait for the next event that matches the filters and print it as JSON.

Without --since, only events after the command starts match. Pass the id of an
event you have already seen as --since to also match anything that happened
since then. The command reconnects if the connection drops, without missing
events. It exits 3 with {"error":"timeout"} when --timeout passes.

--assignee me waits until an issue or pull request is assigned to you (your
LYRA_AGENT name, or your login without one): a work queue. Check
lyra issue list --assignee @me first for work assigned before you started waiting.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{"action": waitActions}
			if waitRepo != "" {
				query.Set("repo", waitRepo)
			}
			if waitAssignee != "" {
				query.Set("assignee", waitAssignee)
			}
			return waitAndEmit(cmd, query, timeout)
		},
	}
	eventWait.Flags().StringVar(&waitRepo, "repo", "", "only events for this repository")
	eventWait.Flags().StringArrayVar(&waitActions, "action", nil, "only these actions, as glob patterns such as push or repo.* (repeatable)")
	eventWait.Flags().StringVar(&waitAssignee, "assignee", "", "only assignments to this login or agent name, or me")
	eventWait.Flags().Int64("since", 0, "match events after this event id (default: from now)")
	eventWait.Flags().DurationVar(&timeout, "timeout", 0, "give up after this long, such as 30m (default: wait forever)")
	event.AddCommand(eventList, eventWait)

	itemCommands := func(use, segment, noun, states string) *cobra.Command {
		parent := &cobra.Command{Use: use, Short: "Manage " + noun + "s"}
		itemsPath := func(repo string) string {
			return repoAPI(repo) + "/" + segment
		}
		var itemTitle, head, base string
		var itemLabels, itemAssignees []string
		createCommand := &cobra.Command{
			Use:   "create <repo>",
			Short: "Open a new " + noun,
			Args:  ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if itemTitle == "" {
					return UsageError{errors.New(use + " create needs --title")}
				}
				body, _, err := readText(cmd, "body")
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
		addTextFlags(createCommand, "body", "body text in Markdown")
		createCommand.Flags().StringArrayVar(&itemLabels, "label", nil, "add this label, which must exist (repeatable)")
		createCommand.Flags().StringArrayVar(&itemAssignees, "assignee", nil, "assign a login or agent name, or @me (repeatable)")

		var listState string
		var listLabels, listAssignees []string
		var listLimit int
		listCommand := &cobra.Command{
			Use:   "list <repo>",
			Short: "List " + noun + "s, newest first",
			Args:  ExactArgs(1),
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
			Args:  ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return call(cmd, http.MethodGet, itemsPath(args[0])+"/"+url.PathEscape(args[1]), nil)
			},
		}

		var commentPath, commentSide string
		var commentLine int64
		commentCommand := &cobra.Command{
			Use:   "comment <repo> <number>",
			Short: "Comment on a " + noun,
			Args:  ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				body, set, err := readText(cmd, "body")
				if err != nil {
					return err
				}
				if !set {
					return UsageError{errors.New(use + " comment needs --body or --body-file")}
				}
				comment := map[string]any{"body": body}
				if commentPath != "" || commentLine != 0 {
					comment["path"], comment["line"], comment["side"] = commentPath, commentLine, commentSide
				}
				return call(cmd, http.MethodPost, itemsPath(args[0])+"/"+url.PathEscape(args[1])+"/comments", comment)
			},
		}
		addTextFlags(commentCommand, "body", "body text in Markdown")
		if segment == "pulls" {
			commentCommand.Long = "Comment on a pull request, or with --path and --line on one line of its diff."
			commentCommand.Flags().StringVar(&commentPath, "path", "", "file the comment is on")
			commentCommand.Flags().Int64Var(&commentLine, "line", 0, "line number in the file, as shown in the diff")
			commentCommand.Flags().StringVar(&commentSide, "side", "new", "new for the head version of the line, old for the base version")
		}

		var editTitle string
		var addLabels, removeLabels, addAssignees, removeAssignees []string
		editCommand := &cobra.Command{
			Use:   "edit <repo> <number>",
			Short: "Change a " + noun + "'s title, body, labels or assignees",
			Args:  ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				edit := map[string]any{}
				if cmd.Flags().Changed("title") {
					edit["title"] = editTitle
				}
				body, set, err := readText(cmd, "body")
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
		addTextFlags(editCommand, "body", "body text in Markdown")
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
				Args:  ExactArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					return call(cmd, http.MethodPatch, itemsPath(args[0])+"/"+url.PathEscape(args[1]), map[string]string{"state": change.state})
				},
			})
		}
		prefix := "issue."
		kinds := []string{"comment", "closed", "reopened", "edited", "labeled", "unlabeled", "assigned", "unassigned"}
		defaults := kinds
		if segment == "pulls" {
			prefix = "pr."
			kinds = append([]string{"review", "checks", "merged", "push"}, kinds...)
			defaults = []string{"review", "comment", "checks", "merged", "closed"}
		}
		var waitFor []string
		var waitTimeout time.Duration
		waitCommand := &cobra.Command{
			Use:   "wait <repo> <number>",
			Short: "Wait until someone else acts on a " + noun,
			Long: `Wait until someone other than you acts on the ` + noun + `, then print that event as JSON,
with the comment or review it carries.

You are the same token, or the same LYRA_AGENT and LYRA_SESSION when a session is
set, so another session of the same agent counts as someone else. Without --since,
the wait starts after your own last action on the ` + noun + `, or after the last event a
wait already returned to you, whichever is later: a reply that arrived since you
last acted returns at once, and waiting again doesn't return it twice. With --since
ID it starts after that event instead. It reconnects if the connection drops, and
exits 3 with {"error":"timeout"} when --timeout passes.

--for chooses the actions, from ` + strings.Join(kinds, ", ") + ` (default: ` + strings.Join(defaults, ", ") + `).`,
			Args: ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				chosen := waitFor
				if len(chosen) == 0 {
					chosen = defaults
				}
				query := url.Values{"repo": {args[0]}, "item": {args[1]}, "exclude_self": {"true"}, "since": {"mine"}}
				for _, kind := range chosen {
					if !slices.Contains(kinds, kind) {
						return UsageError{fmt.Errorf("--for %s: choose from %s", kind, strings.Join(kinds, ", "))}
					}
					if kind == "push" {
						kind = "synchronized"
					}
					query.Add("action", prefix+kind)
				}
				return waitAndEmit(cmd, query, waitTimeout)
			},
		}
		waitCommand.Flags().StringSliceVar(&waitFor, "for", nil, "actions to wait for, comma-separated or repeated")
		waitCommand.Flags().Int64("since", 0, "start after this event id (default: after your last action on it)")
		waitCommand.Flags().DurationVar(&waitTimeout, "timeout", 0, "give up after this long, such as 30m (default: wait forever)")
		parent.AddCommand(createCommand, listCommand, viewCommand, commentCommand, editCommand, waitCommand)
		return parent
	}
	issue := itemCommands("issue", "issues", "issue", "open, closed or all")
	pr := itemCommands("pr", "pulls", "pull request", "open, closed, merged or all")
	pullPath := func(args []string) string {
		return repoAPI(args[0]) + "/pulls/" + url.PathEscape(args[1])
	}
	prCommits := &cobra.Command{
		Use:   "commits <repo> <number>",
		Short: "List a pull request's commits with signature status",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, pullPath(args)+"/commits", nil)
		},
	}
	var filesOnly bool
	prDiff := &cobra.Command{
		Use:   "diff <repo> <number>",
		Short: "Print a pull request's diff, or its changed files with --files",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filesOnly {
				return call(cmd, http.MethodGet, pullPath(args)+"/files", nil)
			}
			return call(cmd, http.MethodGet, pullPath(args)+"/diff", nil)
		},
	}
	prDiff.Flags().BoolVar(&filesOnly, "files", false, "list changed files as JSON instead")
	var squash, fastForward, deleteBranch bool
	var mergeHead, mergeTitle string
	prMerge := &cobra.Command{
		Use:   "merge <repo> <number>",
		Short: "Merge a pull request by squashing or fast-forwarding",
		Long: `Merge a pull request into its base branch.

--squash writes one commit with the pull request's changes, signed by Lyra's key
(see lyra signing-key). --ff moves the base branch to the head commit, keeping its
commits and their signatures. Pass --head with the commit you reviewed to refuse the
merge if the branch has moved since. Conflicts and a moved base exit 1 with HTTP 409.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if squash == fastForward {
				return UsageError{errors.New("pr merge needs exactly one of --squash or --ff")}
			}
			request := map[string]any{"method": "ff", "head_sha": mergeHead, "delete_branch": deleteBranch}
			if squash {
				body, set, err := readText(cmd, "body")
				if err != nil {
					return err
				}
				request["method"], request["title"] = "squash", mergeTitle
				if set {
					request["message"] = body
				}
			}
			return call(cmd, http.MethodPost, pullPath(args)+"/merge", request)
		},
	}
	prMerge.Flags().BoolVar(&squash, "squash", false, "squash into one commit signed by Lyra")
	prMerge.Flags().BoolVar(&fastForward, "ff", false, "fast-forward the base branch to the head")
	prMerge.Flags().BoolVar(&deleteBranch, "delete-branch", false, "delete the head branch after merging")
	prMerge.Flags().StringVar(&mergeHead, "head", "", "merge only if the head branch is at this commit")
	prMerge.Flags().StringVar(&mergeTitle, "title", "", "squash commit title (default: the pull request title and number)")
	addTextFlags(prMerge, "body", "body text in Markdown")
	var approve, requestChanges, commentOnly bool
	var commentsFile string
	prReview := &cobra.Command{
		Use:   "review <repo> <number>",
		Short: "Approve a pull request, request changes, or comment on it",
		Long: `Submit a review of the pull request's current head commit.

--approve and --request-changes count toward branch rules that require approval;
the author can't use them on their own pull request. --request-changes needs a body.
--comments-file reads line comments as JSON: [{"path": "main.go", "line": 12, "side":
"new", "body": "..."}], with lines as numbered in the diff (side defaults to new).`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var state string
			switch {
			case approve && !requestChanges && !commentOnly:
				state = "approve"
			case requestChanges && !approve && !commentOnly:
				state = "request_changes"
			case commentOnly && !approve && !requestChanges:
				state = "comment"
			default:
				return UsageError{errors.New("pr review needs exactly one of --approve, --request-changes or --comment")}
			}
			body, _, err := readText(cmd, "body")
			if err != nil {
				return err
			}
			review := map[string]any{"state": state, "body": body}
			if commentsFile != "" {
				var data []byte
				if commentsFile == "-" {
					data, err = io.ReadAll(cmd.InOrStdin())
				} else {
					data, err = os.ReadFile(commentsFile)
				}
				if err != nil {
					return err
				}
				var comments []map[string]any
				if err := json.Unmarshal(data, &comments); err != nil {
					return UsageError{fmt.Errorf("--comments-file: %w", err)}
				}
				review["comments"] = comments
			}
			return call(cmd, http.MethodPost, pullPath(args)+"/reviews", review)
		},
	}
	prReview.Flags().BoolVar(&approve, "approve", false, "approve the changes")
	prReview.Flags().BoolVar(&requestChanges, "request-changes", false, "request changes before merging")
	prReview.Flags().BoolVar(&commentOnly, "comment", false, "comment without approving")
	prReview.Flags().StringVar(&commentsFile, "comments-file", "", "JSON file with line comments, or - for stdin")
	addTextFlags(prReview, "body", "body text in Markdown")
	pr.AddCommand(prCommits, prDiff, prMerge, prReview)

	signingKey := &cobra.Command{
		Use:   "signing-key",
		Short: "Show the public key Lyra signs merge commits with",
		Long:  "Show the public key Lyra signs merge commits with, and an allowed_signers line for git verify-commit.",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/signing-key", nil)
		},
	}
	root.AddCommand(signingKey)

	label := &cobra.Command{Use: "label", Short: "Manage a repository's labels"}
	labelsPath := func(repo string) string {
		return repoAPI(repo) + "/labels"
	}
	var color, labelDescription string
	labelCreate := &cobra.Command{
		Use:   "create <repo> <name>",
		Short: "Create a label",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPost, labelsPath(args[0]), map[string]string{"name": args[1], "color": color, "description": labelDescription})
		},
	}
	labelCreate.Flags().StringVar(&color, "color", "", "six hex digits, such as d73a4a")
	labelCreate.Flags().StringVar(&labelDescription, "description", "", "label description")
	labelList := &cobra.Command{
		Use:   "list <repo>",
		Short: "List labels",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, labelsPath(args[0]), nil)
		},
	}
	labelDelete := &cobra.Command{
		Use:   "delete <repo> <name>",
		Short: "Delete a label and remove it from every issue",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, labelsPath(args[0])+"/"+url.PathEscape(args[1]), nil)
		},
	}
	label.AddCommand(labelCreate, labelList, labelDelete)

	rule := &cobra.Command{Use: "rule", Short: "Manage branch rules"}
	rulesPath := func(repo string) string {
		return repoAPI(repo) + "/rules"
	}
	var requireSigned, requireApproval, requireChecks, allowForcePush, allowDeletion bool
	var allowedMethods []string
	ruleSet := &cobra.Command{
		Use:   "set <repo> <pattern>",
		Short: "Create or replace the rule for branches matching a pattern",
		Long: `Create or replace the rule for branches matching a pattern, such as main or release/*.

--require-signed refuses pushes that add commits without a verified signature to a
matching branch, and merges of pull requests with such commits (squash merges too).
--require-approval refuses merges until someone other than the author approves the
pull request's current head commit, and while a reviewer's latest review requests
changes. --require-checks refuses merges until the Actions jobs for the pull request's
head commit have all succeeded. A matching branch can't be deleted or force-pushed unless
every matching rule allows it with --allow-deletion or --allow-force-push.
--merge-methods limits how pull requests into a matching branch can be merged. Rules apply
to admins as well. Setting a pattern again replaces its rule.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPost, rulesPath(args[0]), map[string]any{
				"pattern": args[1], "require_signed": requireSigned, "require_approval": requireApproval, "require_checks": requireChecks, "allow_force_push": allowForcePush, "allow_deletion": allowDeletion, "merge_methods": allowedMethods,
			})
		},
	}
	ruleSet.Flags().BoolVar(&requireSigned, "require-signed", false, "require verified signatures on every commit")
	ruleSet.Flags().BoolVar(&requireApproval, "require-approval", false, "require an approving review by someone other than the author")
	ruleSet.Flags().BoolVar(&requireChecks, "require-checks", false, "require every Actions job for the pull request's head commit to succeed")
	ruleSet.Flags().BoolVar(&allowForcePush, "allow-force-push", false, "allow pushes that rewrite a matching branch's history")
	ruleSet.Flags().BoolVar(&allowDeletion, "allow-deletion", false, "allow deleting a matching branch")
	ruleSet.Flags().StringSliceVar(&allowedMethods, "merge-methods", nil, "allowed merge methods, ff and/or squash (default: both)")
	ruleList := &cobra.Command{
		Use:   "list <repo>",
		Short: "List branch rules",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, rulesPath(args[0]), nil)
		},
	}
	ruleDelete := &cobra.Command{
		Use:   "delete <repo> <pattern>",
		Short: "Delete the rule for a pattern",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, rulesPath(args[0])+"?"+url.Values{"pattern": {args[1]}}.Encode(), nil)
		},
	}
	rule.AddCommand(ruleSet, ruleList, ruleDelete)

	root.AddCommand(me, repo, commit, tag, token, key, event, issue, pr, label, rule, releaseCommands(call, emit), imageCommands(call, emit), runnerCommands(call), ownerCommands(call), runCommands(call, emit), workflowCommands(call), secretCommands(call), variableCommands(call), adminCommands(call))
	return root
}

func imageCommands(call func(*cobra.Command, string, string, any) error, emit func(*cobra.Command, []byte) error) *cobra.Command {
	image := &cobra.Command{Use: "image", Short: "List and delete container images"}
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List a repository's image tags with their digests and pull references",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, repoAPI(args[0])+"/images", nil)
		},
	}
	var yes bool
	remove := &cobra.Command{
		Use:   "delete <name>:<tag>|<name>@<digest>",
		Short: "Delete an image tag, or a manifest and every tag pointing at it",
		Long: `Delete an image tag, such as lyra/runner:v1, or a manifest by digest, such as
lyra@sha256:…, which also removes every tag pointing at it. Needs admin access. Layers stay
in storage until registry garbage collection exists.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return UsageError{errors.New("image delete can't be undone; pass --yes")}
			}
			name, reference, found := strings.Cut(args[0], "@")
			if !found {
				colon := strings.LastIndex(args[0], ":")
				if colon < 0 || colon < strings.LastIndex(args[0], "/") {
					return UsageError{fmt.Errorf("%s has no tag or digest", args[0])}
				}
				name, reference = args[0][:colon], args[0][colon+1:]
			}
			if _, err := request(http.MethodDelete, "/v2/"+name+"/manifests/"+url.PathEscape(reference), nil); err != nil {
				return err
			}
			return emit(cmd, []byte(`{"deleted": `+strconv.Quote(args[0])+"}\n"))
		},
	}
	remove.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	image.AddCommand(list, remove)
	return image
}

type releaseInfo struct {
	ID     int64  `json:"id"`
	Tag    string `json:"tag"`
	Draft  bool   `json:"draft"`
	Assets []struct {
		Name   string `json:"name"`
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
	} `json:"assets"`
}

func releaseCommands(call func(*cobra.Command, string, string, any) error, emit func(*cobra.Command, []byte) error) *cobra.Command {
	release := &cobra.Command{Use: "release", Short: "Manage releases and their assets"}
	releasesPath := func(repo string) string {
		return repoAPI(repo) + "/releases"
	}
	find := func(repo, tag string) (releaseInfo, string, error) {
		path := releasesPath(repo) + "/tags/" + strings.ReplaceAll(url.PathEscape(tag), "%2F", "/")
		if tag == "latest" {
			path = releasesPath(repo) + "/latest"
		}
		var found releaseInfo
		output, err := request(http.MethodGet, path, nil)
		if err != nil {
			return found, "", err
		}
		if err := json.Unmarshal(output, &found); err != nil {
			return found, "", err
		}
		return found, releasesPath(repo) + "/" + fmt.Sprint(found.ID), nil
	}

	var target, title string
	var draft, prerelease, generateNotes bool
	var assets []string
	create := &cobra.Command{
		Use:   "create <repo> <tag>",
		Short: "Create a release, uploading its assets before it's published",
		Long: `Create a release for a tag. If the tag doesn't exist, publishing creates it at --target
(default: the default branch). A draft doesn't create the tag until it's published.

With --asset, the release is created as a draft, the assets are uploaded, and then it is
published, so release.published fires once with every asset present. If an upload fails,
the release stays a draft; finish it with lyra release upload and lyra release publish.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, _, err := readText(cmd, "notes")
			if err != nil {
				return err
			}
			for _, file := range assets {
				if _, err := os.Stat(file); err != nil {
					return UsageError{err}
				}
			}
			output, err := request(http.MethodPost, releasesPath(args[0]), map[string]any{
				"tag": args[1], "target": target, "title": title, "notes": notes, "draft": draft || len(assets) > 0, "prerelease": prerelease,
				"generate_notes": generateNotes,
			})
			if err != nil {
				return err
			}
			if len(assets) == 0 {
				return emit(cmd, output)
			}
			var created releaseInfo
			if err := json.Unmarshal(output, &created); err != nil {
				return err
			}
			path := releasesPath(args[0]) + "/" + fmt.Sprint(created.ID)
			for _, file := range assets {
				if _, err := uploadAsset(path, file, false); err != nil {
					return fmt.Errorf("upload %s: %w; the release is still a draft", file, err)
				}
			}
			if draft {
				return call(cmd, http.MethodGet, path, nil)
			}
			return call(cmd, http.MethodPatch, path, map[string]any{"draft": false})
		},
	}
	create.Flags().StringVar(&target, "target", "", "branch or commit to tag if the tag doesn't exist (default: the default branch)")
	create.Flags().StringVar(&title, "title", "", "release title (default: the tag)")
	addTextFlags(create, "notes", "release notes in Markdown")
	create.Flags().BoolVar(&draft, "draft", false, "keep the release as a draft")
	create.Flags().BoolVar(&generateNotes, "generate-notes", false, "append a list of the pull requests merged since the previous release")
	create.Flags().BoolVar(&prerelease, "prerelease", false, "mark as a prerelease, so it's never the latest release")
	create.Flags().StringArrayVar(&assets, "asset", nil, "file to upload as an asset (repeatable)")

	var notesTarget string
	var notesPrerelease bool
	notes := &cobra.Command{
		Use:   "notes <repo> <tag>",
		Short: "Preview the notes --generate-notes would write",
		Long: `Preview the notes --generate-notes would write: the pull requests whose merge commit
is reachable from the tag (or --target, if the tag doesn't exist yet) but not from the
previous release. The previous release is the most recently published one whose commit
is an ancestor; prereleases count only with --prerelease.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{"tag": {args[1]}, "target": {notesTarget}}
			if notesPrerelease {
				query.Set("prerelease", "true")
			}
			return call(cmd, http.MethodGet, releasesPath(args[0])+"/notes?"+query.Encode(), nil)
		},
	}
	notes.Flags().StringVar(&notesTarget, "target", "", "branch or commit, if the tag doesn't exist (default: the default branch)")
	notes.Flags().BoolVar(&notesPrerelease, "prerelease", false, "for a prerelease, so earlier prereleases count as the previous release")
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List releases, drafts first (drafts need write access)",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, releasesPath(args[0]), nil)
		},
	}
	view := &cobra.Command{
		Use:   "view <repo> <tag|latest>",
		Short: "Show a release with its assets and download URLs",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, path, err := find(args[0], args[1])
			if err != nil {
				return err
			}
			return call(cmd, http.MethodGet, path, nil)
		},
	}
	var editTitle, editTarget string
	var editPrerelease bool
	edit := &cobra.Command{
		Use:   "edit <repo> <tag>",
		Short: "Change a release's title, notes or prerelease flag, or a draft's target",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			changes := map[string]any{}
			if cmd.Flags().Changed("title") {
				changes["title"] = editTitle
			}
			if cmd.Flags().Changed("target") {
				changes["target"] = editTarget
			}
			if cmd.Flags().Changed("prerelease") {
				changes["prerelease"] = editPrerelease
			}
			notes, set, err := readText(cmd, "notes")
			if err != nil {
				return err
			}
			if set {
				changes["notes"] = notes
			}
			if len(changes) == 0 {
				return UsageError{errors.New("nothing to change; pass --title, --notes, --notes-file, --prerelease or --target")}
			}
			_, path, err := find(args[0], args[1])
			if err != nil {
				return err
			}
			return call(cmd, http.MethodPatch, path, changes)
		},
	}
	edit.Flags().StringVar(&editTitle, "title", "", "release title")
	edit.Flags().StringVar(&editTarget, "target", "", "branch or commit a draft's tag is created at")
	edit.Flags().BoolVar(&editPrerelease, "prerelease", false, "mark as a prerelease (--prerelease=false to clear)")
	addTextFlags(edit, "notes", "release notes in Markdown")
	publish := &cobra.Command{
		Use:   "publish <repo> <tag>",
		Short: "Publish a draft, creating its tag if needed",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, path, err := find(args[0], args[1])
			if err != nil {
				return err
			}
			return call(cmd, http.MethodPatch, path, map[string]any{"draft": false})
		},
	}
	var cleanupTag, yes bool
	remove := &cobra.Command{
		Use:   "delete <repo> <tag>",
		Short: "Delete a release and its assets; the tag stays unless --cleanup-tag",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return UsageError{errors.New("release delete removes its assets permanently; pass --yes")}
			}
			_, path, err := find(args[0], args[1])
			if err != nil {
				return err
			}
			if cleanupTag {
				path += "?cleanup_tag=true"
			}
			return call(cmd, http.MethodDelete, path, nil)
		},
	}
	remove.Flags().BoolVar(&cleanupTag, "cleanup-tag", false, "delete the tag too")
	remove.Flags().BoolVar(&yes, "yes", false, "confirm permanent deletion")
	var clobber bool
	upload := &cobra.Command{
		Use:   "upload <repo> <tag> <file>...",
		Short: "Upload files as assets of a release, named after the file",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return UsageError{fmt.Errorf("%s takes a repo, a tag and at least one file; see %s --help", cmd.CommandPath(), cmd.CommandPath())}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, path, err := find(args[0], args[1])
			if err != nil {
				return err
			}
			uploaded := []json.RawMessage{}
			for _, file := range args[2:] {
				output, err := uploadAsset(path, file, clobber)
				if err != nil {
					return fmt.Errorf("upload %s: %w", file, err)
				}
				uploaded = append(uploaded, output)
			}
			encoded, err := json.MarshalIndent(uploaded, "", "  ")
			if err != nil {
				return err
			}
			return emit(cmd, append(encoded, '\n'))
		},
	}
	upload.Flags().BoolVar(&clobber, "clobber", false, "replace assets that already exist")
	var pattern, dir string
	download := &cobra.Command{
		Use:   "download <repo> <tag|latest>",
		Short: "Download a release's assets and check their sha256",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := filepath.Match(pattern, ""); err != nil {
				return UsageError{fmt.Errorf("--pattern: %w", err)}
			}
			found, path, err := find(args[0], args[1])
			if err != nil {
				return err
			}
			type downloaded struct {
				Name   string `json:"name"`
				Path   string `json:"path"`
				Size   int64  `json:"size"`
				SHA256 string `json:"sha256"`
			}
			files := []downloaded{}
			for _, asset := range found.Assets {
				if matched, _ := filepath.Match(pattern, asset.Name); pattern != "" && !matched {
					continue
				}
				file := filepath.Join(dir, asset.Name)
				if err := downloadAsset(path+"/assets/"+url.PathEscape(asset.Name), file, asset.SHA256); err != nil {
					return fmt.Errorf("download %s: %w", asset.Name, err)
				}
				files = append(files, downloaded{Name: asset.Name, Path: file, Size: asset.Size, SHA256: asset.SHA256})
			}
			if pattern != "" && len(files) == 0 {
				return fmt.Errorf("no asset of %s matches %s", found.Tag, pattern)
			}
			encoded, err := json.MarshalIndent(files, "", "  ")
			if err != nil {
				return err
			}
			return emit(cmd, append(encoded, '\n'))
		},
	}
	download.Flags().StringVar(&pattern, "pattern", "", "only assets whose name matches this glob")
	download.Flags().StringVar(&dir, "dir", ".", "directory to write the files to")
	deleteAsset := &cobra.Command{
		Use:   "delete-asset <repo> <tag> <name>",
		Short: "Delete one asset of a release",
		Args:  ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, path, err := find(args[0], args[1])
			if err != nil {
				return err
			}
			return call(cmd, http.MethodDelete, path+"/assets/"+url.PathEscape(args[2]), nil)
		},
	}
	release.AddCommand(create, notes, list, view, edit, publish, remove, upload, download, deleteAsset)
	return release
}

func uploadAsset(path, file string, clobber bool) ([]byte, error) {
	handle, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	info, err := handle.Stat()
	if err != nil {
		return nil, err
	}
	query := url.Values{"name": {filepath.Base(file)}}
	if clobber {
		query.Set("clobber", "true")
	}
	req, err := newRequest(context.Background(), http.MethodPost, path+"/assets?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Body = handle
	req.ContentLength = info.Size()
	contentType := mime.TypeByExtension(filepath.Ext(file))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, responseError(response)
	}
	return io.ReadAll(response.Body)
}

func downloadAsset(path, file, sha string) error {
	req, err := newRequest(context.Background(), http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	req.Header.Del("Accept")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return responseError(response)
	}
	temporary, err := os.CreateTemp(filepath.Dir(file), ".lyra-download-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	hash := sha256.New()
	_, err = io.Copy(io.MultiWriter(temporary, hash), response.Body)
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != sha {
		return fmt.Errorf("sha256 is %s, expected %s", got, sha)
	}
	return os.Rename(temporary.Name(), file)
}

func addTextFlags(cmd *cobra.Command, name, usage string) {
	cmd.Flags().String(name, "", usage)
	cmd.Flags().String(name+"-file", "", "read the "+name+" from this file, or - for stdin")
}

func readText(cmd *cobra.Command, name string) (string, bool, error) {
	flags := cmd.Flags()
	if flags.Changed(name) && flags.Changed(name+"-file") {
		return "", false, UsageError{fmt.Errorf("pass --%s or --%s-file, not both", name, name)}
	}
	if flags.Changed(name + "-file") {
		file, _ := flags.GetString(name + "-file")
		var data []byte
		var err error
		if file == "-" {
			data, err = io.ReadAll(cmd.InOrStdin())
		} else {
			data, err = os.ReadFile(file)
		}
		return string(data), true, err
	}
	text, _ := flags.GetString(name)
	return text, flags.Changed(name), nil
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
		Error  string `json:"error"`
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(data, &apiError) == nil && apiError.Error != "" {
		return fmt.Errorf("%s (HTTP %d)", apiError.Error, response.StatusCode)
	}
	if len(apiError.Errors) > 0 {
		return fmt.Errorf("%s: %s (HTTP %d)", apiError.Errors[0].Code, apiError.Errors[0].Message, response.StatusCode)
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

func runnerCommands(call func(*cobra.Command, string, string, any) error) *cobra.Command {
	runner := &cobra.Command{Use: "runner", Short: "Manage Actions runners and their registration tokens"}
	list := &cobra.Command{
		Use:   "list",
		Short: "List registered runners, with labels, version and whether they polled in the last minute",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/runners", nil)
		},
	}
	var runnerName string
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a runner and print its uuid and token, shown once",
		Long: `Create a runner and print its uuid and token, shown once, for a connection in the
forgejo-runner config file instead of forgejo-runner register:

  server:
    connections:
      lyra:
        url: <url>
        uuid: <uuid>
        token: <token>
        labels: [...]

The runner declares its labels and version when it starts. Many processes can share one
runner, for example forgejo-runner one-job in a fresh pod per job.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPost, "/api/runners", map[string]string{"name": runnerName})
		},
	}
	create.Flags().StringVar(&runnerName, "name", "", "runner name (required)")
	remove := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a runner; it can't fetch jobs any more",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, "/api/runners/"+url.PathEscape(args[0]), nil)
		},
	}
	token := &cobra.Command{Use: "token", Short: "Manage runner registration tokens"}
	var name string
	tokenCreate := &cobra.Command{
		Use:   "create",
		Short: "Create a registration token, shown once",
		Long: `Create a registration token, shown once in "secret". Register a runner with it:

  forgejo-runner register --no-interactive --instance <url> --token <secret> --name <name> --labels <labels>

"url" is the address runners reach Lyra at (--actions-url on the server). A token can
register any number of runners, such as one ephemeral runner per job, until it's deleted.
Deleting it doesn't remove the runners it registered.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPost, "/api/runners/tokens", map[string]string{"name": name})
		},
	}
	tokenCreate.Flags().StringVar(&name, "name", "", "what the token is for (required)")
	tokenList := &cobra.Command{
		Use:   "list",
		Short: "List registration tokens",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/runners/tokens", nil)
		},
	}
	tokenDelete := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a registration token",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, "/api/runners/tokens/"+url.PathEscape(args[0]), nil)
		},
	}
	token.AddCommand(tokenCreate, tokenList, tokenDelete)
	runner.AddCommand(create, list, remove, token)
	return runner
}

func repoAPI(name string) string {
	owner, repo, _ := strings.Cut(name, "/")
	return "/api/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
}

func ownerCommands(call func(*cobra.Command, string, string, any) error) *cobra.Command {
	owner := &cobra.Command{Use: "owner", Short: "Manage owners, the namespaces repositories live in"}
	var description string
	create := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an owner, such as krosf, for repositories named <owner>/<name>",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPost, "/api/owners", map[string]string{"name": args[0], "description": description})
		},
	}
	create.Flags().StringVar(&description, "description", "", "owner description")
	list := &cobra.Command{
		Use:   "list",
		Short: "List owners with their repositories",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/owners", nil)
		},
	}
	view := &cobra.Command{
		Use:   "view <name>",
		Short: "Show an owner with its repositories",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/owners/"+url.PathEscape(args[0]), nil)
		},
	}
	remove := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an owner that has no repositories",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, "/api/owners/"+url.PathEscape(args[0]), nil)
		},
	}
	owner.AddCommand(create, list, view, remove)
	return owner
}

type runInfo struct {
	Number int64  `json:"number"`
	Status string `json:"status"`
	Jobs   []struct {
		ID     int64  `json:"id"`
		Key    string `json:"key"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"jobs"`
}

var runFinished = []string{"success", "failure", "cancelled", "skipped"}

func runCommands(call func(*cobra.Command, string, string, any) error, emit func(*cobra.Command, []byte) error) *cobra.Command {
	run := &cobra.Command{Use: "run", Short: "Follow and control Actions runs"}
	runPath := func(repo, number string) string {
		return repoAPI(repo) + "/runs/" + url.PathEscape(number)
	}
	fetch := func(repo, number string) (runInfo, []byte, error) {
		var info runInfo
		output, err := request(http.MethodGet, runPath(repo, number), nil)
		if err != nil {
			return info, nil, err
		}
		return info, output, json.Unmarshal(output, &info)
	}
	var limit int
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List runs, newest first, with their jobs",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, repoAPI(args[0])+"/runs?limit="+fmt.Sprint(limit), nil)
		},
	}
	list.Flags().IntVar(&limit, "limit", 30, "how many runs, up to 100")
	view := &cobra.Command{
		Use:   "view <repo> <number>",
		Short: "Show a run with its jobs and steps",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, runPath(args[0], args[1]), nil)
		},
	}
	var jobFilter string
	var follow bool
	logs := &cobra.Command{
		Use:   "logs <repo> <number>",
		Short: "Print a run's job logs as plain text",
		Long: `Print a run's job logs as plain text. With several jobs, each starts with a "== <key>"
line. --job picks one job by key or id. --follow keeps printing until the jobs finish.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, _, err := fetch(args[0], args[1])
			if err != nil {
				return err
			}
			jobs := info.Jobs
			if jobFilter != "" {
				jobs = jobs[:0]
				for _, job := range info.Jobs {
					if job.Key == jobFilter || fmt.Sprint(job.ID) == jobFilter {
						jobs = append(jobs, job)
					}
				}
				if len(jobs) == 0 {
					return fmt.Errorf("run #%d has no job %s", info.Number, jobFilter)
				}
			}
			out := cmd.OutOrStdout()
			for _, job := range jobs {
				if len(info.Jobs) > 1 && jobFilter == "" {
					fmt.Fprintf(out, "== %s\n", job.Key)
				}
				offset := "0"
				for {
					req, err := newRequest(cmd.Context(), http.MethodGet, runPath(args[0], args[1])+"/jobs/"+fmt.Sprint(job.ID)+"/logs?offset="+offset, nil)
					if err != nil {
						return err
					}
					response, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}
					if response.StatusCode >= 400 {
						defer response.Body.Close()
						return responseError(response)
					}
					_, err = io.Copy(out, response.Body)
					response.Body.Close()
					if err != nil {
						return err
					}
					offset = response.Header.Get("Lyra-Log-Next")
					if !follow || response.Header.Get("Lyra-Log-Done") == "true" {
						break
					}
					time.Sleep(time.Second)
				}
			}
			return nil
		},
	}
	logs.Flags().StringVar(&jobFilter, "job", "", "only this job, by key or id")
	logs.Flags().BoolVar(&follow, "follow", false, "keep printing until the jobs finish")
	var timeout time.Duration
	wait := &cobra.Command{
		Use:   "wait <repo> <number>",
		Short: "Wait for a run to finish; exit 0 if it succeeded, 1 if not, 3 on timeout",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cursor := "0"
			latest, err := request(http.MethodGet, "/api/events?limit=1&repo="+url.QueryEscape(args[0]), nil)
			if err != nil {
				return err
			}
			var events []struct {
				ID int64 `json:"id"`
			}
			if err := json.Unmarshal(latest, &events); err != nil {
				return err
			}
			if len(events) > 0 {
				cursor = fmt.Sprint(events[0].ID)
			}
			deadline := time.Now().Add(timeout)
			for {
				info, output, err := fetch(args[0], args[1])
				if err != nil {
					return err
				}
				if slices.Contains(runFinished, info.Status) {
					if err := emit(cmd, output); err != nil {
						return err
					}
					if info.Status != "success" {
						return fmt.Errorf("run #%d finished: %s", info.Number, info.Status)
					}
					return nil
				}
				remaining := time.Until(deadline)
				if timeout > 0 && remaining <= 0 {
					return ErrTimeout
				}
				query := url.Values{"repo": {args[0]}, "action": {"run.completed"}, "since": {cursor}}
				event, err := waitForEvent("/api/events/stream?"+query.Encode(), max(remaining, 0))
				if err != nil {
					return err
				}
				var completed struct {
					ID int64 `json:"id"`
				}
				if err := json.Unmarshal(event, &completed); err != nil {
					return err
				}
				cursor = fmt.Sprint(completed.ID)
			}
		},
	}
	wait.Flags().DurationVar(&timeout, "timeout", 0, "give up after this long, such as 30m (default: no limit)")
	cancel := &cobra.Command{
		Use:   "cancel <repo> <number>",
		Short: "Cancel a run; running jobs are stopped by their runners",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPost, runPath(args[0], args[1])+"/cancel", nil)
		},
	}
	var failedOnly bool
	rerun := &cobra.Command{
		Use:   "rerun <repo> <number>",
		Short: "Run a finished run's jobs again",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := runPath(args[0], args[1]) + "/rerun"
			if failedOnly {
				path += "?failed=true"
			}
			return call(cmd, http.MethodPost, path, nil)
		},
	}
	rerun.Flags().BoolVar(&failedOnly, "failed", false, "only jobs that didn't succeed, and the jobs that need them")
	run.AddCommand(list, view, logs, wait, cancel, rerun)
	return run
}

func workflowCommands(call func(*cobra.Command, string, string, any) error) *cobra.Command {
	workflow := &cobra.Command{Use: "workflow", Short: "Start Actions workflows"}
	var ref string
	var inputs []string
	run := &cobra.Command{
		Use:   "run <repo> <workflow>",
		Short: "Start a workflow that has a workflow_dispatch trigger",
		Long: `Start a workflow that has a workflow_dispatch trigger, from the workflow file at --ref
(default: the default branch). <workflow> is the file name, such as deploy.yml, or its
path. Inputs are checked against the workflow's: unknown ones are refused, missing ones
take their default, and required ones without a default must be given. Prints the run.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			values := map[string]string{}
			for _, input := range inputs {
				name, value, found := strings.Cut(input, "=")
				if !found || name == "" {
					return UsageError{fmt.Errorf("--input %q: want name=value", input)}
				}
				values[name] = value
			}
			return call(cmd, http.MethodPost, repoAPI(args[0])+"/workflows/dispatch", map[string]any{"workflow": args[1], "ref": ref, "inputs": values})
		},
	}
	run.Flags().StringVar(&ref, "ref", "", "branch or tag to run the workflow from (default: the default branch)")
	run.Flags().StringArrayVar(&inputs, "input", nil, "input as name=value (repeatable)")
	workflow.AddCommand(run)
	return workflow
}

func secretCommands(call func(*cobra.Command, string, string, any) error) *cobra.Command {
	secret := &cobra.Command{Use: "secret", Short: "Manage Actions secrets of a repository (admin)"}
	set := &cobra.Command{
		Use:   "set <repo> <NAME>",
		Short: "Set a secret from stdin; one trailing newline is dropped",
		Long: `Set a secret from stdin, so the value stays out of shell history: for example
lyra secret set krosf/lyra DEPLOY_KEY < key.txt. One trailing newline is dropped. Names are
upper-cased. Jobs read it as ${{ secrets.NAME }}, and runners mask it in logs.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			return call(cmd, http.MethodPut, repoAPI(args[0])+"/secrets/"+url.PathEscape(args[1]), map[string]string{"value": strings.TrimSuffix(string(value), "\n")})
		},
	}
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List secret names, never their values",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, repoAPI(args[0])+"/secrets", nil)
		},
	}
	remove := &cobra.Command{
		Use:   "delete <repo> <NAME>",
		Short: "Delete a secret",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, repoAPI(args[0])+"/secrets/"+url.PathEscape(args[1]), nil)
		},
	}
	secret.AddCommand(set, list, remove)
	return secret
}

func variableCommands(call func(*cobra.Command, string, string, any) error) *cobra.Command {
	variable := &cobra.Command{Use: "variable", Short: "Manage Actions variables of a repository"}
	set := &cobra.Command{
		Use:   "set <repo> <NAME> <value>",
		Short: "Set a variable; jobs read it as ${{ vars.NAME }}",
		Args:  ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodPut, repoAPI(args[0])+"/variables/"+url.PathEscape(args[1]), map[string]string{"value": args[2]})
		},
	}
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List variables with their values",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, repoAPI(args[0])+"/variables", nil)
		},
	}
	remove := &cobra.Command{
		Use:   "delete <repo> <NAME>",
		Short: "Delete a variable",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodDelete, repoAPI(args[0])+"/variables/"+url.PathEscape(args[1]), nil)
		},
	}
	variable.AddCommand(set, list, remove)
	return variable
}

func adminCommands(call func(*cobra.Command, string, string, any) error) *cobra.Command {
	admin := &cobra.Command{Use: "admin", Short: "Backups, read-only mode and status (unrestricted admin)"}
	var output string
	backup := &cobra.Command{
		Use:   "backup",
		Short: "Write a consistent backup of the database and every repository as a tar file",
		Long: `Write a consistent backup as a tar file: lyra.db (made with VACUUM INTO), every bare
repository, and manifest.json (repository refs, and the stored files and images the
database refers to). Writes to Lyra pause while it's taken; the pause is reported.

Signing and secrets keys aren't included, nor are the files in object storage (release
assets, registry layers, job logs): back those up separately. Restore with lyra restore.
The command fails if the tar ends without its manifest, so a cut-off download never
passes as a backup.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := newRequest(cmd.Context(), http.MethodPost, "/api/admin/backup", nil)
			if err != nil {
				return err
			}
			req.Header.Del("Accept")
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			if response.StatusCode >= 400 {
				return responseError(response)
			}
			out := cmd.OutOrStdout()
			var file *os.File
			if output != "" && output != "-" {
				if file, err = os.Create(output); err != nil {
					return err
				}
				defer file.Close()
				out = file
			}
			archive := tar.NewReader(io.TeeReader(response.Body, out))
			last := ""
			for {
				header, err := archive.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return fmt.Errorf("backup stream: %w", err)
				}
				last = header.Name
			}
			if _, err := io.Copy(io.Discard, io.TeeReader(response.Body, out)); err != nil {
				return err
			}
			if last != "manifest.json" {
				return errors.New("the backup ended without its manifest; it's incomplete")
			}
			if file != nil {
				if err := file.Sync(); err != nil {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "{\"file\":%q,\"pause_ms\":%s}\n", output, response.Header.Get("Lyra-Backup-Pause-Ms"))
			}
			return nil
		},
	}
	backup.Flags().StringVar(&output, "output", "", "file to write (default: stdout)")
	readOnly := &cobra.Command{
		Use:   "readonly on|off|status",
		Short: "Refuse every write with 503 while reads keep working; a restart turns it off",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "on", "off":
				return call(cmd, http.MethodPost, "/api/admin/readonly", map[string]bool{"enabled": args[0] == "on"})
			case "status":
				return call(cmd, http.MethodGet, "/api/admin/status", nil)
			}
			return UsageError{fmt.Errorf("readonly takes on, off or status, not %q", args[0])}
		},
	}
	status := &cobra.Command{
		Use:   "status",
		Short: "Show read-only mode and the last backup",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return call(cmd, http.MethodGet, "/api/admin/status", nil)
		},
	}
	storageCheck := &cobra.Command{
		Use:   "storage-check",
		Short: "List stored files and images the database refers to but storage lacks; exit 1 if any",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := request(http.MethodGet, "/api/admin/storage-check", nil)
			if err != nil {
				return err
			}
			var result struct {
				Missing []json.RawMessage `json:"missing"`
			}
			if err := json.Unmarshal(output, &result); err != nil {
				return err
			}
			if _, err := cmd.OutOrStdout().Write(output); err != nil {
				return err
			}
			if len(result.Missing) > 0 {
				return fmt.Errorf("%d stored objects are missing", len(result.Missing))
			}
			return nil
		},
	}
	admin.AddCommand(backup, readOnly, status, storageCheck)
	return admin
}
