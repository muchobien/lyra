package cli

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

type ownerCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

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
	var body ownerCreateRequest
	create := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an owner, such as krosf, for repositories named <owner>/<name>",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body.Name = args[0]
			return app.call(cmd, http.MethodPost, "/api/owners", body)
		},
	}
	create.Flags().StringVar(&body.Description, "description", "", "owner description")
	return create
}

func (app *app) ownerListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List owners with their repositories",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, "/api/owners", nil)
		},
	}
}

func (app *app) ownerViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view <name>",
		Short: "Show an owner with its repositories",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, "/api/owners/"+url.PathEscape(args[0]), nil)
		},
	}
}

func (app *app) ownerDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an owner that has no repositories",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodDelete, "/api/owners/"+url.PathEscape(args[0]), nil)
		},
	}
}
