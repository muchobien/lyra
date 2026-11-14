package cli

import (
	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) ruleCommand() *cobra.Command {
	rule := &cobra.Command{Use: "rule", Short: "Manage branch rules"}
	rule.AddCommand(app.ruleSetCommand(), app.ruleListCommand(), app.ruleDeleteCommand())
	return rule
}

func (app *app) ruleSetCommand() *cobra.Command {
	var requireSigned, requireApproval, requireChecks, allowForcePush, allowDeletion bool
	var methods []string
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
			mergeMethods := []lyraapi.SetRuleRequestMergeMethods{}
			for _, method := range methods {
				mergeMethods = append(mergeMethods, lyraapi.SetRuleRequestMergeMethods(method))
			}
			body := lyraapi.SetRuleRequest{
				Pattern:         args[1],
				RequireSigned:   &requireSigned,
				RequireApproval: &requireApproval,
				RequireChecks:   &requireChecks,
				AllowForcePush:  &allowForcePush,
				AllowDeletion:   &allowDeletion,
				MergeMethods:    &mergeMethods,
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.SetRule(cmd.Context(), owner, repo, body))
		},
	}
	flags := set.Flags()
	flags.BoolVar(&requireSigned, "require-signed", false, "require verified signatures on every commit")
	flags.BoolVar(
		&requireApproval,
		"require-approval",
		false,
		"require an approving review by someone other than the author",
	)
	flags.BoolVar(
		&requireChecks,
		"require-checks",
		false,
		"require every Actions job for the pull request's head commit to succeed",
	)
	flags.BoolVar(
		&allowForcePush,
		"allow-force-push",
		false,
		"allow pushes that rewrite a matching branch's history",
	)
	flags.BoolVar(&allowDeletion, "allow-deletion", false, "allow deleting a matching branch")
	flags.StringSliceVar(
		&methods,
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
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListRules(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) ruleDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <repo> <pattern>",
		Short: "Delete the rule for a pattern",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			params := lyraapi.DeleteRuleParams{Pattern: args[1]}
			return app.show(cmd)(app.client.api.DeleteRule(cmd.Context(), owner, repo, &params))
		},
	}
}
