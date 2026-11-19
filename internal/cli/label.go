package cli

import (
	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) labelCommand() *cobra.Command {
	label := &cobra.Command{Use: "label", Short: "Manage a repository's labels"}
	label.AddCommand(app.repoScoped(app.labelCreateCommand(), app.labelListCommand(), app.labelDeleteCommand())...)
	return label
}

func (app *app) labelCreateCommand() *cobra.Command {
	var color, description string
	create := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a label",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			body := lyraapi.CreateLabelRequest{Name: args[1], Color: &color, Description: &description}
			return app.show(cmd)(app.client.api.CreateLabel(cmd.Context(), owner, repo, body))
		},
	}
	create.Flags().StringVarP(&color, "color", "c", "", "six hex digits, such as d73a4a")
	create.Flags().StringVarP(&description, "description", "d", "", "label description")
	return create
}

func (app *app) labelListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List labels",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListLabels(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) labelDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a label and remove it from every issue",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.DeleteLabel(cmd.Context(), owner, repo, args[1]))
		},
	}
}
