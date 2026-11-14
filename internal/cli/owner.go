package cli

import (
	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) ownerCommand() *cobra.Command {
	owner := &cobra.Command{Use: "owner", Short: "Manage owners, the namespaces repositories live in"}
	owner.AddCommand(
		app.ownerCreateCommand(),
		app.ownerListCommand(),
		app.ownerViewCommand(),
		app.ownerDeleteCommand(),
	)
	return owner
}

func (app *app) ownerCreateCommand() *cobra.Command {
	var description string
	create := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an owner, such as krosf, for repositories named <owner>/<name>",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := lyraapi.CreateOwnerRequest{Name: args[0], Description: &description}
			return app.show(cmd)(app.client.api.CreateOwner(cmd.Context(), body))
		},
	}
	create.Flags().StringVar(&description, "description", "", "owner description")
	return create
}

func (app *app) ownerListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List owners with their repositories",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.show(cmd)(app.client.api.ListOwners(cmd.Context()))
		},
	}
}

func (app *app) ownerViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view <name>",
		Short: "Show an owner with its repositories",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.show(cmd)(app.client.api.GetOwner(cmd.Context(), args[0]))
		},
	}
}

func (app *app) ownerDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an owner that has no repositories",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.show(cmd)(app.client.api.DeleteOwner(cmd.Context(), args[0]))
		},
	}
}
