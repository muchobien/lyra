package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type itemKind struct {
	use            string
	segment        string
	noun           string
	states         string
	prefix         string
	actions        []string
	defaultActions []string
}

type itemCreateRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels"`
	Assignees []string `json:"assignees"`
	*pullBranches
}

type pullBranches struct {
	Head string `json:"head"`
	Base string `json:"base"`
}

type itemCommentRequest struct {
	Body string `json:"body"`
	*lineLocation
}

type lineLocation struct {
	Path string `json:"path"`
	Line int64  `json:"line"`
	Side string `json:"side"`
}

type itemEditRequest struct {
	Title           *string  `json:"title,omitzero"`
	Body            *string  `json:"body,omitzero"`
	AddLabels       []string `json:"add_labels,omitempty"`
	RemoveLabels    []string `json:"remove_labels,omitempty"`
	AddAssignees    []string `json:"add_assignees,omitempty"`
	RemoveAssignees []string `json:"remove_assignees,omitempty"`
}

type itemStateRequest struct {
	State string `json:"state"`
}

var itemActions = []string{
	"comment",
	"closed",
	"reopened",
	"edited",
	"labeled",
	"unlabeled",
	"assigned",
	"unassigned",
}

func (kind itemKind) isPull() bool {
	return kind.segment == "pulls"
}

func (kind itemKind) listPath(repo string) string {
	return repoAPI(repo) + "/" + kind.segment
}

func (kind itemKind) itemPath(repo, number string) string {
	return kind.listPath(repo) + "/" + url.PathEscape(number)
}

func (app *app) issueCommand() *cobra.Command {
	return app.itemCommand(itemKind{
		use:            "issue",
		segment:        "issues",
		noun:           "issue",
		states:         "open, closed or all",
		prefix:         "issue.",
		actions:        itemActions,
		defaultActions: itemActions,
	})
}

func (app *app) itemCommand(kind itemKind) *cobra.Command {
	parent := &cobra.Command{Use: kind.use, Short: "Manage " + kind.noun + "s"}
	parent.AddCommand(
		app.itemStateCommand(kind, "close <repo> <number>", "Close a "+kind.noun, "closed"),
		app.itemStateCommand(kind, "reopen <repo> <number>", "Reopen a closed "+kind.noun, "open"),
		app.itemCreateCommand(kind),
		app.itemListCommand(kind),
		app.itemViewCommand(kind),
		app.itemCommentCommand(kind),
		app.itemEditCommand(kind),
		app.itemWaitCommand(kind),
	)
	return parent
}

func (app *app) itemCreateCommand(kind itemKind) *cobra.Command {
	var body itemCreateRequest
	var branches pullBranches
	create := &cobra.Command{
		Use:   "create <repo>",
		Short: "Open a new " + kind.noun,
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if body.Title == "" {
				return UsageError{errors.New(kind.use + " create needs --title")}
			}
			text, _, err := readText(cmd, "body")
			if err != nil {
				return err
			}
			body.Body = text
			if kind.isPull() {
				if branches.Head == "" {
					return UsageError{errors.New("pr create needs --head")}
				}
				body.pullBranches = &branches
			}
			return app.call(cmd, http.MethodPost, kind.listPath(args[0]), body)
		},
	}
	flags := create.Flags()
	flags.StringVar(&body.Title, "title", "", "title (required)")
	if kind.isPull() {
		flags.StringVar(&branches.Head, "head", "", "branch with the changes (required)")
		flags.StringVar(&branches.Base, "base", "", "branch to merge into (default: the default branch)")
	}
	addTextFlags(create, "body", "body text in Markdown")
	flags.StringArrayVar(&body.Labels, "label", nil, "add this label, which must exist (repeatable)")
	flags.StringArrayVar(&body.Assignees, "assignee", nil, "assign a login or agent name, or @me (repeatable)")
	return create
}

func (app *app) itemListCommand(kind itemKind) *cobra.Command {
	var state string
	var labels, assignees []string
	var limit int
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List " + kind.noun + "s, newest first",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{
				"state":    {state},
				"limit":    {fmt.Sprint(limit)},
				"label":    labels,
				"assignee": assignees,
			}
			return app.call(cmd, http.MethodGet, kind.listPath(args[0])+"?"+query.Encode(), nil)
		},
	}
	flags := list.Flags()
	flags.StringVar(&state, "state", "open", kind.states)
	flags.StringArrayVar(&labels, "label", nil, "only "+kind.noun+"s with this label (repeatable)")
	flags.StringArrayVar(
		&assignees,
		"assignee",
		nil,
		"only "+kind.noun+"s assigned to this login or agent, or @me (repeatable)",
	)
	flags.IntVar(&limit, "limit", 30, "maximum number of "+kind.noun+"s")
	return list
}

func (app *app) itemViewCommand(kind itemKind) *cobra.Command {
	return &cobra.Command{
		Use:   "view <repo> <number>",
		Short: "Show a " + kind.noun + " with its comments",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, kind.itemPath(args[0], args[1]), nil)
		},
	}
}

func (app *app) itemCommentCommand(kind itemKind) *cobra.Command {
	var location lineLocation
	comment := &cobra.Command{
		Use:   "comment <repo> <number>",
		Short: "Comment on a " + kind.noun,
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, set, err := readText(cmd, "body")
			if err != nil {
				return err
			}
			if !set {
				return UsageError{errors.New(kind.use + " comment needs --body or --body-file")}
			}
			body := itemCommentRequest{Body: text}
			if location.Path != "" || location.Line != 0 {
				body.lineLocation = &location
			}
			return app.call(cmd, http.MethodPost, kind.itemPath(args[0], args[1])+"/comments", body)
		},
	}
	addTextFlags(comment, "body", "body text in Markdown")
	if kind.isPull() {
		comment.Long = "Comment on a pull request, or with --path and --line on one line of its diff."
		flags := comment.Flags()
		flags.StringVar(&location.Path, "path", "", "file the comment is on")
		flags.Int64Var(&location.Line, "line", 0, "line number in the file, as shown in the diff")
		flags.StringVar(
			&location.Side,
			"side",
			"new",
			"new for the head version of the line, old for the base version",
		)
	}
	return comment
}

func (app *app) itemEditCommand(kind itemKind) *cobra.Command {
	var title string
	var body itemEditRequest
	edit := &cobra.Command{
		Use:   "edit <repo> <number>",
		Short: "Change a " + kind.noun + "'s title, body, labels or assignees",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("title") {
				body.Title = &title
			}
			text, set, err := readText(cmd, "body")
			if err != nil {
				return err
			}
			if set {
				body.Body = &text
			}
			labels := len(body.AddLabels) + len(body.RemoveLabels)
			assignees := len(body.AddAssignees) + len(body.RemoveAssignees)
			if body.Title == nil && body.Body == nil && labels+assignees == 0 {
				help := "lyra " + kind.use + " edit --help"
				return UsageError{errors.New(kind.use + " edit needs at least one change; see " + help)}
			}
			return app.call(cmd, http.MethodPatch, kind.itemPath(args[0], args[1]), body)
		},
	}
	flags := edit.Flags()
	flags.StringVar(&title, "title", "", "new title")
	addTextFlags(edit, "body", "body text in Markdown")
	flags.StringArrayVar(&body.AddLabels, "add-label", nil, "add a label (repeatable)")
	flags.StringArrayVar(&body.RemoveLabels, "remove-label", nil, "remove a label (repeatable)")
	flags.StringArrayVar(
		&body.AddAssignees,
		"add-assignee",
		nil,
		"assign a login or agent name, or @me (repeatable)",
	)
	flags.StringArrayVar(
		&body.RemoveAssignees,
		"remove-assignee",
		nil,
		"unassign a login or agent name, or @me (repeatable)",
	)
	return edit
}

func (app *app) itemStateCommand(kind itemKind, use, short, state string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := itemStateRequest{State: state}
			return app.call(cmd, http.MethodPatch, kind.itemPath(args[0], args[1]), body)
		},
	}
}

func (app *app) itemWaitCommand(kind itemKind) *cobra.Command {
	var chosen []string
	var timeout time.Duration
	wait := &cobra.Command{
		Use:   "wait <repo> <number>",
		Short: "Wait until someone else acts on a " + kind.noun,
		Long: `Wait until someone other than you acts on the ` + kind.noun + `, then print that event as JSON,
with the comment or review it carries.

You are the same token, or the same LYRA_AGENT and LYRA_SESSION when a session is
set, so another session of the same agent counts as someone else. Without --since,
the wait starts after your own last action on the ` + kind.noun + `, or after the last event a
wait already returned to you, whichever is later: a reply that arrived since you
last acted returns at once, and waiting again doesn't return it twice. With --since
ID it starts after that event instead. It reconnects if the connection drops, and
exits 3 with {"error":"timeout"} when --timeout passes.

With --for checks, a pull request whose head has no Actions runs at all returns at once
with state none.

--for chooses the actions, from ` + strings.Join(kind.actions, ", ") +
			` (default: ` + strings.Join(kind.defaultActions, ", ") + `).`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.waitForItem(cmd, kind, args, chosen, timeout)
		},
	}
	flags := wait.Flags()
	flags.StringSliceVar(&chosen, "for", nil, "actions to wait for, comma-separated or repeated")
	flags.Int64("since", 0, "start after this event id (default: after your last action on it)")
	flags.DurationVar(&timeout, "timeout", 0, "give up after this long, such as 30m (default: wait forever)")
	return wait
}

func (app *app) waitForItem(
	cmd *cobra.Command,
	kind itemKind,
	args []string,
	chosen []string,
	timeout time.Duration,
) error {
	actions := chosen
	if len(actions) == 0 {
		actions = kind.defaultActions
	}
	query := url.Values{"repo": {args[0]}, "item": {args[1]}, "exclude_self": {"true"}, "since": {"mine"}}
	for _, action := range actions {
		if !slices.Contains(kind.actions, action) {
			choices := strings.Join(kind.actions, ", ")
			return UsageError{fmt.Errorf("--for %s: choose from %s", action, choices)}
		}
		if action == "push" {
			action = "synchronized"
		}
		query.Add("action", kind.prefix+action)
	}
	if kind.isPull() && slices.Contains(chosen, "checks") {
		event, err := app.missingChecks(cmd, args)
		if err != nil {
			return err
		}
		if event != nil {
			return app.emitIndented(cmd, event)
		}
	}
	return app.waitAndEmit(cmd, query, timeout)
}
