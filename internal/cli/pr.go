package cli

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

type pullChecks struct {
	SHA   string `json:"sha"`
	State string `json:"state"`
}

type checksEvent struct {
	Action string     `json:"action"`
	Data   pullChecks `json:"data"`
	Item   int64      `json:"item"`
	Repo   string     `json:"repo"`
}

type mergeRequest struct {
	Method       string  `json:"method"`
	HeadSHA      string  `json:"head_sha"`
	DeleteBranch bool    `json:"delete_branch"`
	Title        *string `json:"title,omitzero"`
	Message      *string `json:"message,omitzero"`
}

type reviewRequest struct {
	State    string           `json:"state"`
	Body     string           `json:"body"`
	Comments []map[string]any `json:"comments,omitzero"`
}

type reviewOptions struct {
	approve        bool
	requestChanges bool
	commentOnly    bool
	commentsFile   string
}

func pullPath(args []string) string {
	return repoAPI(args[0]) + "/pulls/" + url.PathEscape(args[1])
}

func (app *app) prCommand() *cobra.Command {
	pr := app.itemCommand(itemKind{
		use:            "pr",
		segment:        "pulls",
		noun:           "pull request",
		states:         "open, closed, merged or all",
		prefix:         "pr.",
		actions:        append([]string{"review", "checks", "merged", "push"}, itemActions...),
		defaultActions: []string{"review", "comment", "checks", "merged", "closed"},
	})
	pr.AddCommand(
		app.prCommitsCommand(),
		app.prDiffCommand(),
		app.prMergeCommand(),
		app.prReviewCommand(),
	)
	return pr
}

func (app *app) missingChecks(cmd *cobra.Command, args []string) (*checksEvent, error) {
	output, err := app.client.request(cmd.Context(), http.MethodGet, pullPath(args), nil)
	if err != nil {
		return nil, err
	}
	var pull struct {
		Number int64      `json:"number"`
		Checks pullChecks `json:"checks"`
	}
	if err := json.Unmarshal(output, &pull); err != nil {
		return nil, err
	}
	if pull.Checks.State != "none" {
		return nil, nil
	}
	return &checksEvent{
		Action: "pr.checks",
		Data:   pullChecks{SHA: pull.Checks.SHA, State: "none"},
		Item:   pull.Number,
		Repo:   args[0],
	}, nil
}

func (app *app) prCommitsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "commits <repo> <number>",
		Short: "List a pull request's commits with signature status",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, pullPath(args)+"/commits", nil)
		},
	}
}

func (app *app) prDiffCommand() *cobra.Command {
	var filesOnly bool
	diff := &cobra.Command{
		Use:   "diff <repo> <number>",
		Short: "Print a pull request's diff, or its changed files with --files",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filesOnly {
				return app.call(cmd, http.MethodGet, pullPath(args)+"/files", nil)
			}
			return app.call(cmd, http.MethodGet, pullPath(args)+"/diff", nil)
		},
	}
	diff.Flags().BoolVar(&filesOnly, "files", false, "list changed files as JSON instead")
	return diff
}

func (app *app) prMergeCommand() *cobra.Command {
	var squash, fastForward bool
	var title string
	body := mergeRequest{Method: "ff"}
	merge := &cobra.Command{
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
			if squash {
				message, set, err := readText(cmd, "body")
				if err != nil {
					return err
				}
				body.Method, body.Title = "squash", &title
				if set {
					body.Message = &message
				}
			}
			return app.call(cmd, http.MethodPost, pullPath(args)+"/merge", body)
		},
	}
	flags := merge.Flags()
	flags.BoolVar(&squash, "squash", false, "squash into one commit signed by Lyra")
	flags.BoolVar(&fastForward, "ff", false, "fast-forward the base branch to the head")
	flags.BoolVar(&body.DeleteBranch, "delete-branch", false, "delete the head branch after merging")
	flags.StringVar(&body.HeadSHA, "head", "", "merge only if the head branch is at this commit")
	flags.StringVar(&title, "title", "", "squash commit title (default: the pull request title and number)")
	addTextFlags(merge, "body", "body text in Markdown")
	return merge
}

func (app *app) prReviewCommand() *cobra.Command {
	var options reviewOptions
	review := &cobra.Command{
		Use:   "review <repo> <number>",
		Short: "Approve a pull request, request changes, or comment on it",
		Long: `Submit a review of the pull request's current head commit.

--approve and --request-changes count toward branch rules that require approval;
the author can't use them on their own pull request. --request-changes needs a body.
--comments-file reads line comments as JSON: [{"path": "main.go", "line": 12, "side":
"new", "body": "..."}], with lines as numbered in the diff (side defaults to new).`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.submitReview(cmd, args, options)
		},
	}
	flags := review.Flags()
	flags.BoolVar(&options.approve, "approve", false, "approve the changes")
	flags.BoolVar(&options.requestChanges, "request-changes", false, "request changes before merging")
	flags.BoolVar(&options.commentOnly, "comment", false, "comment without approving")
	flags.StringVar(&options.commentsFile, "comments-file", "", "JSON file with line comments, or - for stdin")
	addTextFlags(review, "body", "body text in Markdown")
	return review
}

func (options reviewOptions) state() (string, error) {
	switch {
	case options.approve && !options.requestChanges && !options.commentOnly:
		return "approve", nil
	case options.requestChanges && !options.approve && !options.commentOnly:
		return "request_changes", nil
	case options.commentOnly && !options.approve && !options.requestChanges:
		return "comment", nil
	}
	return "", UsageError{errors.New("pr review needs exactly one of --approve, --request-changes or --comment")}
}

func (app *app) submitReview(cmd *cobra.Command, args []string, options reviewOptions) error {
	state, err := options.state()
	if err != nil {
		return err
	}
	text, _, err := readText(cmd, "body")
	if err != nil {
		return err
	}
	review := reviewRequest{State: state, Body: text}
	if options.commentsFile != "" {
		data, err := readInput(cmd, options.commentsFile)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &review.Comments); err != nil {
			return UsageError{fmt.Errorf("--comments-file: %w", err)}
		}
	}
	return app.call(cmd, http.MethodPost, pullPath(args)+"/reviews", review)
}
