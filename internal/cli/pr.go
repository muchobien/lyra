package cli

import (
	"encoding/json/v2"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
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

type reviewOptions struct {
	approve        bool
	requestChanges bool
	commentOnly    bool
	commentsFile   string
}

func (app *app) prCommand() *cobra.Command {
	pr := app.itemCommand(itemKind{
		use:            "pr",
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

func (app *app) missingChecks(cmd *cobra.Command, repo string, number int64) (*checksEvent, error) {
	owner, name := splitRepo(repo)
	pull, _, err := decode[lyraapi.PullView](app.client.api.GetPull(cmd.Context(), owner, name, number))
	if err != nil {
		return nil, err
	}
	if pull.Checks.State != "none" {
		return nil, nil
	}
	return &checksEvent{
		Action: "pr.checks",
		Data:   pullChecks{SHA: pull.Checks.Sha, State: "none"},
		Item:   pull.Number,
		Repo:   repo,
	}, nil
}

func (app *app) prCommitsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "commits <repo> <number>",
		Short: "List a pull request's commits with signature status",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListPullCommits(cmd.Context(), owner, repo, number))
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
			number, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			if filesOnly {
				return app.show(cmd)(app.client.api.ListPullFiles(cmd.Context(), owner, repo, number))
			}
			return app.show(cmd)(app.client.api.GetPullDiff(cmd.Context(), owner, repo, number))
		},
	}
	diff.Flags().BoolVar(&filesOnly, "files", false, "list changed files as JSON instead")
	return diff
}

func (app *app) prMergeCommand() *cobra.Command {
	var squash, fastForward, deleteBranch bool
	var title, head string
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
			number, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			body := lyraapi.MergeRequest{
				Method:       lyraapi.MergeRequestMethodFf,
				HeadSha:      &head,
				DeleteBranch: &deleteBranch,
			}
			if squash {
				message, set, err := readText(cmd, "body")
				if err != nil {
					return err
				}
				body.Method, body.Title = lyraapi.MergeRequestMethodSquash, &title
				if set {
					body.Message = &message
				}
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.MergePull(cmd.Context(), owner, repo, number, body))
		},
	}
	flags := merge.Flags()
	flags.BoolVar(&squash, "squash", false, "squash into one commit signed by Lyra")
	flags.BoolVar(&fastForward, "ff", false, "fast-forward the base branch to the head")
	flags.BoolVar(&deleteBranch, "delete-branch", false, "delete the head branch after merging")
	flags.StringVar(&head, "head", "", "merge only if the head branch is at this commit")
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

func (options reviewOptions) state() (lyraapi.ReviewRequestState, error) {
	switch {
	case options.approve && !options.requestChanges && !options.commentOnly:
		return lyraapi.ReviewRequestStateApprove, nil
	case options.requestChanges && !options.approve && !options.commentOnly:
		return lyraapi.ReviewRequestStateRequestChanges, nil
	case options.commentOnly && !options.approve && !options.requestChanges:
		return lyraapi.ReviewRequestStateComment, nil
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
	number, err := parseNumber(args[1])
	if err != nil {
		return err
	}
	review := lyraapi.ReviewRequest{State: state, Body: &text}
	if options.commentsFile != "" {
		data, err := readInput(cmd, options.commentsFile)
		if err != nil {
			return err
		}
		comments := []lyraapi.LineComment{}
		if err := json.Unmarshal(data, &comments); err != nil {
			return UsageError{fmt.Errorf("--comments-file: %w", err)}
		}
		review.Comments = &comments
	}
	owner, repo := splitRepo(args[0])
	return app.show(cmd)(app.client.api.ReviewPull(cmd.Context(), owner, repo, number, review))
}
