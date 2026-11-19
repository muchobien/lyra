package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

type itemKind struct {
	use            string
	noun           string
	target         string
	states         string
	prefix         string
	actions        []string
	defaultActions []string
}

const attachHelp = `--attach uploads an image or video and links it from the body. A reference to the file
already in the body, such as ![alt](./login.png), is rewritten to point at the upload; other
attachments are appended to the body. Paths in a --body-file are relative to its directory,
then to the working directory. Alt text follows the path after #, as in
--attach './login.png#The login error state', and defaults to the file name; a video renders
as a player and has no alt text. Up to 50 files. If some uploads fail, the command goes ahead
with the rest, prints its result and exits 1.`

var markdownDestination = regexp.MustCompile(`(\]\(\s*<?)([^)\s>]+)`)

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

func (kind itemKind) targetArgs() cobra.PositionalArgs {
	if kind.isPull() {
		return MaximumArgs(1)
	}
	return ExactArgs(1)
}

func (app *app) itemNumber(cmd *cobra.Command, kind itemKind, args []string) (int64, error) {
	if kind.isPull() {
		return app.pullNumber(cmd, args)
	}
	return parseNumber(args[1])
}

func (app *app) issueCommand() *cobra.Command {
	return app.itemCommand(itemKind{
		use:            "issue",
		noun:           "issue",
		target:         "<number>",
		states:         "open, closed or all",
		prefix:         "issue.",
		actions:        itemActions,
		defaultActions: itemActions,
	})
}

func (app *app) itemCommand(kind itemKind) *cobra.Command {
	parent := &cobra.Command{Use: kind.use, Short: "Manage " + kind.noun + "s"}
	parent.AddCommand(app.repoScoped(
		app.itemStateCommand(kind, "close "+kind.target, "Close a "+kind.noun, lyraapi.Closed),
		app.itemStateCommand(kind, "reopen "+kind.target, "Reopen a closed "+kind.noun, lyraapi.Open),
		app.itemCreateCommand(kind),
		app.itemListCommand(kind),
		app.itemViewCommand(kind),
		app.itemCommentCommand(kind),
		app.itemEditCommand(kind),
		app.itemWaitCommand(kind),
	)...)
	return parent
}

func (app *app) itemCreateCommand(kind itemKind) *cobra.Command {
	var title, head, base string
	var labels, assignees, attachments []string
	create := &cobra.Command{
		Use:   "create",
		Short: "Open a new " + kind.noun,
		Long:  "Open a new " + kind.noun + ".\n\n" + attachHelp,
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return UsageError{errors.New(kind.use + " create needs --title")}
			}
			text, _, err := readText(cmd, "body")
			if err != nil {
				return err
			}
			text, attachErr := app.attach(cmd, args[0], text, attachments)
			if _, usage := errors.AsType[UsageError](attachErr); usage {
				return attachErr
			}
			owner, repo := splitRepo(args[0])
			if !kind.isPull() {
				body := lyraapi.CreateIssueRequest{Title: title, Body: &text, Labels: &labels, Assignees: &assignees}
				response, err := app.client.api.CreateIssue(cmd.Context(), owner, repo, body)
				return errors.Join(app.show(cmd)(response, err), attachErr)
			}
			if head == "" {
				if head, err = currentBranch(cmd); err != nil {
					return err
				}
			}
			body := lyraapi.CreatePullRequest{
				Title:     title,
				Body:      &text,
				Head:      head,
				Base:      &base,
				Labels:    &labels,
				Assignees: &assignees,
			}
			created, output, err := decode[lyraapi.Pull](app.client.api.CreatePull(cmd.Context(), owner, repo, body))
			if err != nil {
				return errors.Join(err, attachErr)
			}
			if err := app.emit(cmd, output); err != nil {
				return err
			}
			printWaitHint(cmd, args[0], created.Number)
			return attachErr
		},
	}
	flags := create.Flags()
	flags.StringVarP(&title, "title", "t", "", "title (required)")
	if kind.isPull() {
		flags.StringVarP(&head, "head", "H", "", "branch with the changes (default: the current branch)")
		flags.StringVarP(&base, "base", "B", "", "branch to merge into (default: the default branch)")
	}
	addTextFlags(create, "body", "body text in Markdown")
	flags.StringArrayVarP(&labels, "label", "l", nil, "add this label, which must exist (repeatable)")
	flags.StringArrayVarP(&assignees, "assignee", "a", nil, "assign a login or agent name, or @me (repeatable)")
	flags.StringArrayVar(&attachments, "attach", nil, "attach an image or video, as <file>#<alt text> (repeatable)")
	return create
}

func (app *app) itemListCommand(kind itemKind) *cobra.Command {
	var state, head, base string
	var labels, assignees []string
	var limit int64
	list := &cobra.Command{
		Use:   "list",
		Short: "List " + kind.noun + "s, newest first",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			if kind.isPull() {
				params := lyraapi.ListPullsParams{
					State:    &state,
					Label:    &labels,
					Assignee: &assignees,
					Limit:    &limit,
					Head:     &head,
					Base:     &base,
				}
				return app.show(cmd)(app.client.api.ListPulls(cmd.Context(), owner, repo, &params))
			}
			params := lyraapi.ListIssuesParams{State: &state, Label: &labels, Assignee: &assignees, Limit: &limit}
			return app.show(cmd)(app.client.api.ListIssues(cmd.Context(), owner, repo, &params))
		},
	}
	flags := list.Flags()
	flags.StringVarP(&state, "state", "s", "open", kind.states)
	flags.StringArrayVarP(&labels, "label", "l", nil, "only "+kind.noun+"s with this label (repeatable)")
	flags.StringArrayVarP(
		&assignees,
		"assignee",
		"a",
		nil,
		"only "+kind.noun+"s assigned to this login or agent, or @me (repeatable)",
	)
	flags.Int64VarP(&limit, "limit", "L", 30, "maximum number of "+kind.noun+"s")
	if kind.isPull() {
		flags.StringVarP(&head, "head", "H", "", "only pull requests from this branch")
		flags.StringVarP(&base, "base", "B", "", "only pull requests into this branch")
	}
	return list
}

func (app *app) itemViewCommand(kind itemKind) *cobra.Command {
	return &cobra.Command{
		Use:   "view " + kind.target,
		Short: "Show a " + kind.noun + " with its comments",
		Args:  kind.targetArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := app.itemNumber(cmd, kind, args)
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			if kind.isPull() {
				response, err := app.client.api.GetPull(cmd.Context(), owner, repo, number)
				pull, output, err := decode[lyraapi.PullView](response, err)
				if err != nil {
					return err
				}
				if err := app.emit(cmd, output); err != nil {
					return err
				}
				if pull.State == "open" {
					printWaitHint(cmd, args[0], number)
				}
				return nil
			}
			return app.show(cmd)(app.client.api.GetIssue(cmd.Context(), owner, repo, number))
		},
	}
}

func (app *app) itemCommentCommand(kind itemKind) *cobra.Command {
	var path, side string
	var line int64
	var attachments []string
	comment := &cobra.Command{
		Use:   "comment " + kind.target,
		Short: "Comment on a " + kind.noun,
		Long:  "Comment on a " + kind.noun + ".\n\n" + attachHelp,
		Args:  kind.targetArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, set, err := readText(cmd, "body")
			if err != nil {
				return err
			}
			if !set && len(attachments) == 0 {
				return UsageError{errors.New(kind.use + " comment needs --body, --body-file or --attach")}
			}
			number, err := app.itemNumber(cmd, kind, args)
			if err != nil {
				return err
			}
			text, attachErr := app.attach(cmd, args[0], text, attachments)
			if _, usage := errors.AsType[UsageError](attachErr); usage {
				return attachErr
			}
			owner, repo := splitRepo(args[0])
			body := lyraapi.CommentRequest{Body: text}
			if !kind.isPull() {
				response, err := app.client.api.CommentIssue(cmd.Context(), owner, repo, number, body)
				return errors.Join(app.show(cmd)(response, err), attachErr)
			}
			if path != "" || line != 0 {
				lineSide := lyraapi.CommentRequestSide(side)
				body.Path, body.Line, body.Side = &path, &line, &lineSide
			}
			response, err := app.client.api.CommentPull(cmd.Context(), owner, repo, number, body)
			return errors.Join(app.show(cmd)(response, err), attachErr)
		},
	}
	addTextFlags(comment, "body", "body text in Markdown")
	comment.Flags().StringArrayVar(
		&attachments,
		"attach",
		nil,
		"attach an image or video, as <file>#<alt text> (repeatable)",
	)
	if kind.isPull() {
		comment.Long = "Comment on a pull request, or with --path and --line on one line of its diff.\n\n" + attachHelp
		flags := comment.Flags()
		flags.StringVar(&path, "path", "", "file the comment is on")
		flags.Int64Var(&line, "line", 0, "line number in the file, as shown in the diff")
		flags.StringVar(&side, "side", "new", "new for the head version of the line, old for the base version")
	}
	return comment
}

func (app *app) editItem(cmd *cobra.Command, kind itemKind, args []string, body lyraapi.EditItemRequest) error {
	number, err := app.itemNumber(cmd, kind, args)
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
	var addLabels, removeLabels, addAssignees, removeAssignees, attachments []string
	edit := &cobra.Command{
		Use:   "edit " + kind.target,
		Short: "Change a " + kind.noun + "'s title, body, labels or assignees",
		Long: "Change a " + kind.noun + "'s title, body, labels or assignees.\n\n" + attachHelp +
			"\n\nWithout --body or --body-file, --attach appends to the current body.",
		Args: kind.targetArgs(),
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
			if body.Title == nil && body.Body == nil && labels+assignees+len(attachments) == 0 {
				help := "lyra " + kind.use + " edit --help"
				return UsageError{errors.New(kind.use + " edit needs at least one change; see " + help)}
			}
			if len(attachments) == 0 {
				return app.editItem(cmd, kind, args, body)
			}
			if !set {
				if text, err = app.itemBody(cmd, kind, args); err != nil {
					return err
				}
			}
			text, attachErr := app.attach(cmd, args[0], text, attachments)
			if _, usage := errors.AsType[UsageError](attachErr); usage {
				return attachErr
			}
			body.Body = &text
			return errors.Join(app.editItem(cmd, kind, args, body), attachErr)
		},
	}
	flags := edit.Flags()
	flags.StringVarP(&title, "title", "t", "", "new title")
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
	flags.StringArrayVar(&attachments, "attach", nil, "attach an image or video, as <file>#<alt text> (repeatable)")
	return edit
}

func (app *app) itemBody(cmd *cobra.Command, kind itemKind, args []string) (string, error) {
	number, err := app.itemNumber(cmd, kind, args)
	if err != nil {
		return "", err
	}
	owner, repo := splitRepo(args[0])
	if kind.isPull() {
		pull, _, err := decode[lyraapi.PullView](app.client.api.GetPull(cmd.Context(), owner, repo, number))
		return pull.Body, err
	}
	issue, _, err := decode[lyraapi.ItemView](app.client.api.GetIssue(cmd.Context(), owner, repo, number))
	return issue.Body, err
}

func (app *app) attach(cmd *cobra.Command, repo, text string, files []string) (string, error) {
	if len(files) > 50 {
		return text, UsageError{fmt.Errorf("--attach takes at most 50 files, got %d", len(files))}
	}
	bases := []string{"."}
	if bodyFile, _ := cmd.Flags().GetString("body-file"); bodyFile != "" && bodyFile != "-" {
		bases = []string{filepath.Dir(bodyFile), "."}
	}
	failures := []error{}
	for _, value := range files {
		file, alt, _ := strings.Cut(value, "#")
		attachment, err := app.client.uploadAttachment(cmd.Context(), repo, file)
		if err != nil {
			failures = append(failures, fmt.Errorf("attach %s: %w", file, err))
			continue
		}
		attached, _ := os.Stat(file)
		referenced := false
		text = markdownDestination.ReplaceAllStringFunc(text, func(match string) string {
			parts := markdownDestination.FindStringSubmatch(match)
			for _, base := range bases {
				target := parts[2]
				if !filepath.IsAbs(target) {
					target = filepath.Join(base, target)
				}
				if info, err := os.Stat(target); err == nil && os.SameFile(info, attached) {
					referenced = true
					return parts[1] + attachment.Url
				}
			}
			return match
		})
		if referenced {
			continue
		}
		if strings.HasPrefix(attachment.ContentType, "video/") {
			alt = ""
		} else if alt == "" {
			alt = filepath.Base(file)
		}
		if text != "" {
			text = strings.TrimRight(text, "\n") + "\n\n"
		}
		text += "![" + strings.NewReplacer("[", `\[`, "]", `\]`).Replace(alt) + "](" + attachment.Url + ")"
	}
	return text, errors.Join(failures...)
}

func (app *app) itemStateCommand(kind itemKind, use, short string, state lyraapi.EditItemRequestState) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  kind.targetArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.editItem(cmd, kind, args, lyraapi.EditItemRequest{State: &state})
		},
	}
}

func (app *app) itemWaitCommand(kind itemKind) *cobra.Command {
	var chosen []string
	var timeout time.Duration
	wait := &cobra.Command{
		Use:   "wait " + kind.target,
		Short: "Wait until someone else acts on a " + kind.noun,
		Long: `Wait until someone other than you acts on the ` + kind.noun + `, then print that event as JSON,
with the comment or review it carries.

You are the same token, or the same LYRA_AGENT and LYRA_SESSION when a session is
set, so another session of the same agent counts as someone else. Without --since,
the wait starts after your own last action on the ` + kind.noun + `, or after the last event a
wait already returned to you, whichever is later: a reply that arrived since you
last acted returns at once, and waiting again doesn't return it twice. With --since
ID it starts after that event instead. It reconnects if the connection drops, and
exits 8 with {"error":"timeout"} when --timeout passes. --count N prints N events and
--follow keeps printing until stopped, one JSON object per line.

With --for checks, a pull request whose head has no Actions runs at all returns at once
with state none.

--for chooses the actions, from ` + strings.Join(kind.actions, ", ") +
			` (default: ` + strings.Join(kind.defaultActions, ", ") + `).`,
		Args: kind.targetArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.waitForItem(cmd, kind, args, chosen, timeout)
		},
	}
	flags := wait.Flags()
	flags.StringSliceVar(&chosen, "for", nil, "actions to wait for, comma-separated or repeated")
	flags.Int64("since", 0, "start after this event id (default: after your last action on it)")
	addTimeoutFlag(wait, &timeout)
	flags.Int("count", 1, "print this many events before exiting")
	flags.Bool("follow", false, "keep printing events until stopped")
	return wait
}

func (app *app) waitForItem(
	cmd *cobra.Command,
	kind itemKind,
	args []string,
	chosen []string,
	timeout time.Duration,
) error {
	number, err := app.itemNumber(cmd, kind, args)
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
