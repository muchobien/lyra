package cli

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

type variableSetRequest struct {
	Value string `json:"value"`
}

func variablePath(repo, name string) string {
	return repoAPI(repo) + "/variables/" + url.PathEscape(name)
}

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
			body := variableSetRequest{Value: args[2]}
			return app.call(cmd, http.MethodPut, variablePath(args[0], args[1]), body)
		},
	}
}

func (app *app) variableListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo>",
		Short: "List variables with their values",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, repoAPI(args[0])+"/variables", nil)
		},
	}
}

func (app *app) variableDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <repo> <NAME>",
		Short: "Delete a variable",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodDelete, variablePath(args[0], args[1]), nil)
		},
	}
}
