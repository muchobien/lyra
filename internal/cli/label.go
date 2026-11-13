package cli

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

type labelCreateRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func labelsPath(repo string) string {
	return repoAPI(repo) + "/labels"
}

func (app *app) labelCommand() *cobra.Command {
	label := &cobra.Command{Use: "label", Short: "Manage a repository's labels"}
	label.AddCommand(app.labelCreateCommand(), app.labelListCommand(), app.labelDeleteCommand())
	return label
}

func (app *app) labelCreateCommand() *cobra.Command {
	var body labelCreateRequest
	create := &cobra.Command{
		Use:   "create <repo> <name>",
		Short: "Create a label",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body.Name = args[1]
			return app.call(cmd, http.MethodPost, labelsPath(args[0]), body)
		},
	}
	create.Flags().StringVar(&body.Color, "color", "", "six hex digits, such as d73a4a")
	create.Flags().StringVar(&body.Description, "description", "", "label description")
	return create
}

func (app *app) labelListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo>",
		Short: "List labels",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, labelsPath(args[0]), nil)
		},
	}
}

func (app *app) labelDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <repo> <name>",
		Short: "Delete a label and remove it from every issue",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodDelete, labelsPath(args[0])+"/"+url.PathEscape(args[1]), nil)
		},
	}
}
