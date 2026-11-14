package cli

import (
	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) commitCommand() *cobra.Command {
	commit := &cobra.Command{Use: "commit", Short: "Read commits and their signature status"}
	commit.AddCommand(app.commitListCommand(), app.commitShowCommand())
	return commit
}

func (app *app) commitListCommand() *cobra.Command {
	var ref string
	var limit int64
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List commits with signature status",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := lyraapi.ListCommitsParams{Limit: &limit}
			if ref != "" {
				params.Ref = &ref
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListCommits(cmd.Context(), owner, repo, &params))
		},
	}
	list.Flags().StringVar(&ref, "ref", "", "branch, tag or commit (default: the default branch)")
	list.Flags().Int64Var(&limit, "limit", 30, "maximum number of commits")
	return list
}

func (app *app) commitShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <repo> <rev>",
		Short: "Show one commit, verified with git verify-commit",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.GetCommit(cmd.Context(), owner, repo, args[1]))
		},
	}
}
