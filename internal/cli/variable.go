package cli

import (
	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) variableCommand() *cobra.Command {
	variable := &cobra.Command{Use: "variable", Short: "Manage Actions variables of a repository"}
	variable.AddCommand(app.repoScoped(
		app.variableSetCommand(),
		app.variableListCommand(),
		app.variableDeleteCommand(),
	)...)
	return variable
}

func (app *app) variableSetCommand() *cobra.Command {
	set := &cobra.Command{
		Use:   "set <NAME>",
		Short: "Set a variable from --body or stdin; jobs read it as ${{ vars.NAME }}",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := readValue(cmd)
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			body := lyraapi.ValueRequest{Value: value}
			return app.show(cmd)(app.client.api.SetVariable(cmd.Context(), owner, repo, args[1], body))
		},
	}
	set.Flags().StringP("body", "b", "", "the value (default: read from stdin)")
	return set
}

func (app *app) variableListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List variables with their values",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListVariables(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) variableDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <NAME>",
		Short: "Delete a variable",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.DeleteVariable(cmd.Context(), owner, repo, args[1]))
		},
	}
}
