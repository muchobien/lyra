package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) repoCommand() *cobra.Command {
	repo := &cobra.Command{Use: "repo", Short: "Manage repositories"}
	repo.AddCommand(
		app.repoCreateCommand(),
		app.repoListCommand(),
		app.repoGetCommand(),
		app.repoDeleteCommand(),
	)
	return repo
}

func (app *app) repoCreateCommand() *cobra.Command {
	var description string
	create := &cobra.Command{
		Use:   "create <owner>/<name>",
		Short: "Create a repository under an existing owner",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := lyraapi.CreateRepoRequest{Name: args[0], Description: &description}
			return app.show(cmd)(app.client.api.CreateRepo(cmd.Context(), body))
		},
	}
	create.Flags().StringVar(&description, "description", "", "repository description")
	return create
}

func (app *app) repoListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.show(cmd)(app.client.api.ListRepos(cmd.Context()))
		},
	}
}

func (app *app) repoGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Show a repository with its branches and clone URL",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.GetRepo(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) repoDeleteCommand() *cobra.Command {
	var yes bool
	remove := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a repository permanently",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				message := "repo delete removes the repository permanently; pass --yes"
				return UsageError{errors.New(message)}
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.DeleteRepo(cmd.Context(), owner, repo))
		},
	}
	remove.Flags().BoolVar(&yes, "yes", false, "confirm permanent deletion")
	return remove
}
