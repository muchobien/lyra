package cli

import (
	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) variableCommand() *cobra.Command {
	variable := &cobra.Command{Use: "variable", Short: "Manage Actions variables of a repository"}
	variable.AddCommand(app.variableSetCommand(), app.variableListCommand(), app.variableDeleteCommand())
	return variable
}

func (app *app) variableSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <repo> <NAME> <value>",
		Short: "Set a variable; jobs read it as ${{ vars.NAME }}",
		Args:  ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			body := lyraapi.ValueRequest{Value: args[2]}
			return app.show(cmd)(app.client.api.SetVariable(cmd.Context(), owner, repo, args[1], body))
		},
	}
}

func (app *app) variableListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo>",
		Short: "List variables with their values",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListVariables(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) variableDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <repo> <NAME>",
		Short: "Delete a variable",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.DeleteVariable(cmd.Context(), owner, repo, args[1]))
		},
	}
}
