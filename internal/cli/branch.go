package cli

import (
	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) branchCommand() *cobra.Command {
	branch := &cobra.Command{Use: "branch", Short: "List and delete branches"}
	branch.AddCommand(app.repoScoped(app.branchListCommand(), app.branchDeleteCommand())...)
	return branch
}

func (app *app) branchListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List branches with their last commit and distance from the default branch",
		Long: `List branches: the default branch first, then the most recently updated. Each has its
last commit, ahead and behind (commits it has that the default branch doesn't, and the
reverse), and protected when a branch rule forbids deleting it.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListBranches(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) branchDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <branch>",
		Short: "Delete a branch",
		Long: `Delete a branch, as git push --delete would: branch rules apply, and pull requests and
workflows see the deletion. The default branch can't be deleted. Needs write access.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			params := lyraapi.DeleteBranchParams{Name: args[1]}
			return app.show(cmd)(app.client.api.DeleteBranch(cmd.Context(), owner, repo, &params))
		},
	}
}
