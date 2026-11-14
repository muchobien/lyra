package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

type itemKind struct {
	use            string
	noun           string
	states         string
	prefix         string
	actions        []string
	defaultActions []string
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
	return kind.use == "pr"
}

func (app *app) issueCommand() *cobra.Command {
	return app.itemCommand(itemKind{
		use:            "issue",
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
		app.itemStateCommand(kind, "close <repo> <number>", "Close a "+kind.noun, lyraapi.Closed),
		app.itemStateCommand(kind, "reopen <repo> <number>", "Reopen a closed "+kind.noun, lyraapi.Open),
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
	var title, head, base string
	var labels, assignees []string
	create := &cobra.Command{
		Use:   "create <repo>",
		Short: "Open a new " + kind.noun,
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return UsageError{errors.New(kind.use + " create needs --title")}
			}
			text, _, err := readText(cmd, "body")
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			if !kind.isPull() {
				body := lyraapi.CreateIssueRequest{Title: title, Body: &text, Labels: &labels, Assignees: &assignees}
				return app.show(cmd)(app.client.api.CreateIssue(cmd.Context(), owner, repo, body))
			}
			if head == "" {
				return UsageError{errors.New("pr create needs --head")}
			}
			body := lyraapi.CreatePullRequest{
				Title:     title,
				Body:      &text,
				Head:      head,
				Base:      &base,
				Labels:    &labels,
				Assignees: &assignees,
			}
			return app.show(cmd)(app.client.api.CreatePull(cmd.Context(), owner, repo, body))
		},
	}
	flags := create.Flags()
	flags.StringVar(&title, "title", "", "title (required)")
	if kind.isPull() {
		flags.StringVar(&head, "head", "", "branch with the changes (required)")
		flags.StringVar(&base, "base", "", "branch to merge into (default: the default branch)")
	}
	addTextFlags(create, "body", "body text in Markdown")
	flags.StringArrayVar(&labels, "label", nil, "add this label, which must exist (repeatable)")
	flags.StringArrayVar(&assignees, "assignee", nil, "assign a login or agent name, or @me (repeatable)")
	return create
}

func (app *app) itemListCommand(kind itemKind) *cobra.Command {
	var state string
	var labels, assignees []string
	var limit int64
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List " + kind.noun + "s, newest first",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			if kind.isPull() {
				params := lyraapi.ListPullsParams{State: &state, Label: &labels, Assignee: &assignees, Limit: &limit}
				return app.show(cmd)(app.client.api.ListPulls(cmd.Context(), owner, repo, &params))
			}
			params := lyraapi.ListIssuesParams{State: &state, Label: &labels, Assignee: &assignees, Limit: &limit}
			return app.show(cmd)(app.client.api.ListIssues(cmd.Context(), owner, repo, &params))
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
	flags.Int64Var(&limit, "limit", 30, "maximum number of "+kind.noun+"s")
	return list
}

func (app *app) itemViewCommand(kind itemKind) *cobra.Command {
	return &cobra.Command{
		Use:   "view <repo> <number>",
		Short: "Show a " + kind.noun + " with its comments",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			if kind.isPull() {
				return app.show(cmd)(app.client.api.GetPull(cmd.Context(), owner, repo, number))
			}
			return app.show(cmd)(app.client.api.GetIssue(cmd.Context(), owner, repo, number))
		},
	}
}

func (app *app) itemCommentCommand(kind itemKind) *cobra.Command {
	var path, side string
	var line int64
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
			number, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			body := lyraapi.CommentRequest{Body: text}
			if !kind.isPull() {
				return app.show(cmd)(app.client.api.CommentIssue(cmd.Context(), owner, repo, number, body))
			}
			if path != "" || line != 0 {
				lineSide := lyraapi.CommentRequestSide(side)
				body.Path, body.Line, body.Side = &path, &line, &lineSide
			}
			return app.show(cmd)(app.client.api.CommentPull(cmd.Context(), owner, repo, number, body))
		},
	}
	addTextFlags(comment, "body", "body text in Markdown")
	if kind.isPull() {
		comment.Long = "Comment on a pull request, or with --path and --line on one line of its diff."
		flags := comment.Flags()
		flags.StringVar(&path, "path", "", "file the comment is on")
		flags.Int64Var(&line, "line", 0, "line number in the file, as shown in the diff")
		flags.StringVar(&side, "side", "new", "new for the head version of the line, old for the base version")
	}
	return comment
}

func (app *app) editItem(cmd *cobra.Command, kind itemKind, args []string, body lyraapi.EditItemRequest) error {
	number, err := parseNumber(args[1])
	if err != nil {
		return err
	}
	owner, repo := splitRepo(args[0])
	if kind.isPull() {
		return app.show(cmd)(app.client.api.EditPull(cmd.Context(), owner, repo, number, body))
	}
	return app.show(cmd)(app.client.api.EditIssue(cmd.Context(), owner, repo, number, body))
}

func (app *app) itemEditCommand(kind itemKind) *cobra.Command {
	var title string
	var addLabels, removeLabels, addAssignees, removeAssignees []string
	edit := &cobra.Command{
		Use:   "edit <repo> <number>",
		Short: "Change a " + kind.noun + "'s title, body, labels or assignees",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := lyraapi.EditItemRequest{
				AddLabels:       &addLabels,
				RemoveLabels:    &removeLabels,
				AddAssignees:    &addAssignees,
				RemoveAssignees: &removeAssignees,
			}
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
			labels := len(addLabels) + len(removeLabels)
			assignees := len(addAssignees) + len(removeAssignees)
			if body.Title == nil && body.Body == nil && labels+assignees == 0 {
				help := "lyra " + kind.use + " edit --help"
				return UsageError{errors.New(kind.use + " edit needs at least one change; see " + help)}
			}
			return app.editItem(cmd, kind, args, body)
		},
	}
	flags := edit.Flags()
	flags.StringVar(&title, "title", "", "new title")
	addTextFlags(edit, "body", "body text in Markdown")
	flags.StringArrayVar(&addLabels, "add-label", nil, "add a label (repeatable)")
	flags.StringArrayVar(&removeLabels, "remove-label", nil, "remove a label (repeatable)")
	flags.StringArrayVar(&addAssignees, "add-assignee", nil, "assign a login or agent name, or @me (repeatable)")
	flags.StringArrayVar(
		&removeAssignees,
		"remove-assignee",
		nil,
		"unassign a login or agent name, or @me (repeatable)",
	)
	return edit
}

func (app *app) itemStateCommand(kind itemKind, use, short string, state lyraapi.EditItemRequestState) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.editItem(cmd, kind, args, lyraapi.EditItemRequest{State: &state})
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
	number, err := parseNumber(args[1])
	if err != nil {
		return err
	}
	actions := chosen
	if len(actions) == 0 {
		actions = kind.defaultActions
	}
	patterns := []string{}
	for _, action := range actions {
		if !slices.Contains(kind.actions, action) {
			choices := strings.Join(kind.actions, ", ")
			return UsageError{fmt.Errorf("--for %s: choose from %s", action, choices)}
		}
		if action == "push" {
			action = "synchronized"
		}
		patterns = append(patterns, kind.prefix+action)
	}
	if kind.isPull() && slices.Contains(chosen, "checks") {
		event, err := app.missingChecks(cmd, args[0], number)
		if err != nil {
			return err
		}
		if event != nil {
			return app.emitIndented(cmd, event)
		}
	}
	excludeSelf, since := true, "mine"
	params := lyraapi.StreamEventsParams{
		Repo:        &args[0],
		Item:        &number,
		Action:      &patterns,
		ExcludeSelf: &excludeSelf,
		Since:       &since,
	}
	return app.waitAndEmit(cmd, params, timeout)
}
