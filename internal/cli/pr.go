package cli

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
		target:         "[<number> | <branch>]",
		states:         "open, closed, merged or all",
		prefix:         "pr.",
		actions:        append([]string{"review", "checks", "merged", "push"}, itemActions...),
		defaultActions: []string{"review", "comment", "checks", "merged", "closed"},
	})
	pr.Long = `Manage pull requests.

Commands that act on one pull request take its number or its head branch. Without
either, they act on the open pull request whose head is the current git branch.`
	pr.AddCommand(app.repoScoped(
		app.prChecksCommand(),
		app.prCommitsCommand(),
		app.prDiffCommand(),
		app.prMergeCommand(),
		app.prReviewCommand(),
	)...)
	return pr
}

func printWaitHint(cmd *cobra.Command, repo string, number int64) {
	fmt.Fprintf(cmd.ErrOrStderr(), `hint: to wait for this pull request instead of polling, run one of:
    lyra pr wait %[2]d -R %[1]s --timeout 30m
    lyra pr checks %[2]d -R %[1]s --watch
  - pr wait prints the next review, comment, merge or close by someone else as JSON and exits;
    run it again for the one after. It returns at once for replies that came since you last acted.
  - pr checks --watch exits 0 when the checks pass and 1 when one fails.
  - Both exit 8 when --timeout passes.
`, repo, number)
}

func (app *app) pullNumber(cmd *cobra.Command, args []string) (int64, error) {
	var branch string
	if len(args) > 1 {
		number, err := strconv.ParseInt(strings.TrimPrefix(args[1], "#"), 10, 64)
		if err == nil && number > 0 {
			return number, nil
		}
		branch = args[1]
	} else {
		current, err := currentBranch(cmd)
		if err != nil {
			return 0, err
		}
		branch = current
	}
	owner, repo := splitRepo(args[0])
	state, limit := "open", int64(1)
	params := lyraapi.ListPullsParams{State: &state, Head: &branch, Limit: &limit}
	pulls, _, err := decode[[]lyraapi.Pull](app.client.api.ListPulls(cmd.Context(), owner, repo, &params))
	if err != nil {
		return 0, err
	}
	if len(pulls) == 0 {
		return 0, fmt.Errorf("no open pull request in %s has head %s", args[0], branch)
	}
	return pulls[0].Number, nil
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

func (app *app) prChecksCommand() *cobra.Command {
	var watch bool
	checks := &cobra.Command{
		Use:   "checks [<number> | <branch>]",
		Short: "Show the Actions checks of a pull request's head",
		Long: `Show the Actions checks of a pull request's head commit: its state (success, failure,
pending or none) and its jobs. Exit 0 when they passed, 1 when one failed or none ran,
and 8 while they're pending. --watch waits until they finish.`,
		Args: MaximumArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := app.pullNumber(cmd, args)
			if err != nil {
				return err
			}
			return app.showChecks(cmd, args[0], number, watch)
		},
	}
	checks.Flags().BoolVar(&watch, "watch", false, "wait until the checks finish")
	return checks
}

func (app *app) showChecks(cmd *cobra.Command, repo string, number int64, watch bool) error {
	owner, name := splitRepo(repo)
	for {
		cursor, err := app.latestEvent(cmd, repo)
		if err != nil {
			return err
		}
		pull, _, err := decode[lyraapi.PullView](app.client.api.GetPull(cmd.Context(), owner, name, number))
		if err != nil {
			return err
		}
		if !watch || pull.Checks.State != "pending" {
			if err := app.emitIndented(cmd, pull.Checks); err != nil {
				return err
			}
			switch pull.Checks.State {
			case "success":
				return nil
			case "pending":
				return ErrPending
			case "none":
				return fmt.Errorf("no checks ran on %s", pull.Checks.Sha)
			}
			return fmt.Errorf("checks failed on %s", pull.Checks.Sha)
		}
		actions := []string{"pr.checks", "pr.synchronized"}
		params := lyraapi.StreamEventsParams{Repo: &repo, Item: &number, Action: &actions, Since: &cursor}
		if _, err := app.client.waitForEvent(cmd.Context(), params, 0); err != nil {
			return err
		}
	}
}

func (app *app) prCommitsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "commits [<number> | <branch>]",
		Short: "List a pull request's commits with signature status",
		Args:  MaximumArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := app.pullNumber(cmd, args)
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
		Use:   "diff [<number> | <branch>]",
		Short: "Print a pull request's diff, or its changed files with --files",
		Args:  MaximumArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := app.pullNumber(cmd, args)
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
		Use:   "merge [<number> | <branch>]",
		Short: "Merge a pull request by squashing or fast-forwarding",
		Long: `Merge a pull request into its base branch.

--squash writes one commit with the pull request's changes, signed by Lyra's key
(see lyra signing-key). --ff moves the base branch to the head commit, keeping its
commits and their signatures. Pass --match-head-commit with the commit you reviewed to
refuse the merge if the branch has moved since. Conflicts and a moved base exit 1 with HTTP 409.`,
		Args: MaximumArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if squash == fastForward {
				return UsageError{errors.New("pr merge needs exactly one of --squash or --ff")}
			}
			number, err := app.pullNumber(cmd, args)
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
	flags.BoolVarP(&squash, "squash", "s", false, "squash into one commit signed by Lyra")
	flags.BoolVar(&fastForward, "ff", false, "fast-forward the base branch to the head")
	flags.BoolVarP(&deleteBranch, "delete-branch", "d", false, "delete the head branch after merging")
	flags.StringVar(&head, "match-head-commit", "", "merge only if the head branch is at this commit")
	flags.StringVarP(&title, "subject", "t", "", "squash commit subject (default: the pull request title and number)")
	addTextFlags(merge, "body", "body text in Markdown")
	return merge
}

func (app *app) prReviewCommand() *cobra.Command {
	var options reviewOptions
	review := &cobra.Command{
		Use:   "review [<number> | <branch>]",
		Short: "Approve a pull request, request changes, or comment on it",
		Long: `Submit a review of the pull request's current head commit.

--approve and --request-changes count toward branch rules that require approval;
the author can't use them on their own pull request. --request-changes needs a body.
--comments-file reads line comments as JSON: [{"path": "main.go", "line": 12, "side":
"new", "body": "..."}], with lines as numbered in the diff (side defaults to new).`,
		Args: MaximumArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.submitReview(cmd, args, options)
		},
	}
	flags := review.Flags()
	flags.BoolVarP(&options.approve, "approve", "a", false, "approve the changes")
	flags.BoolVarP(&options.requestChanges, "request-changes", "r", false, "request changes before merging")
	flags.BoolVarP(&options.commentOnly, "comment", "c", false, "comment without approving")
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
	number, err := app.pullNumber(cmd, args)
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
