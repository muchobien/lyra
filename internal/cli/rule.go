package cli

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

type ruleSetRequest struct {
	Pattern         string   `json:"pattern"`
	RequireSigned   bool     `json:"require_signed"`
	RequireApproval bool     `json:"require_approval"`
	RequireChecks   bool     `json:"require_checks"`
	AllowForcePush  bool     `json:"allow_force_push"`
	AllowDeletion   bool     `json:"allow_deletion"`
	MergeMethods    []string `json:"merge_methods"`
}

func rulesPath(repo string) string {
	return repoAPI(repo) + "/rules"
}

func (app *app) ruleCommand() *cobra.Command {
	rule := &cobra.Command{Use: "rule", Short: "Manage branch rules"}
	rule.AddCommand(app.ruleSetCommand(), app.ruleListCommand(), app.ruleDeleteCommand())
	return rule
}

func (app *app) ruleSetCommand() *cobra.Command {
	var body ruleSetRequest
	set := &cobra.Command{
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
			body.Pattern = args[1]
			return app.call(cmd, http.MethodPost, rulesPath(args[0]), body)
		},
	}
	flags := set.Flags()
	flags.BoolVar(&body.RequireSigned, "require-signed", false, "require verified signatures on every commit")
	flags.BoolVar(
		&body.RequireApproval,
		"require-approval",
		false,
		"require an approving review by someone other than the author",
	)
	flags.BoolVar(
		&body.RequireChecks,
		"require-checks",
		false,
		"require every Actions job for the pull request's head commit to succeed",
	)
	flags.BoolVar(
		&body.AllowForcePush,
		"allow-force-push",
		false,
		"allow pushes that rewrite a matching branch's history",
	)
	flags.BoolVar(&body.AllowDeletion, "allow-deletion", false, "allow deleting a matching branch")
	flags.StringSliceVar(
		&body.MergeMethods,
		"merge-methods",
		nil,
		"allowed merge methods, ff and/or squash (default: both)",
	)
	return set
}

func (app *app) ruleListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo>",
		Short: "List branch rules",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, rulesPath(args[0]), nil)
		},
	}
}

func (app *app) ruleDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <repo> <pattern>",
		Short: "Delete the rule for a pattern",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{"pattern": {args[1]}}
			return app.call(cmd, http.MethodDelete, rulesPath(args[0])+"?"+query.Encode(), nil)
		},
	}
}
