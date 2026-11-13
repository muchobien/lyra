package cli

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

type tokenCreateRequest struct {
	Name      string   `json:"name"`
	Scope     string   `json:"scope"`
	Repos     []string `json:"repos"`
	ExpiresIn string   `json:"expires_in"`
}

func (app *app) tokenCommand() *cobra.Command {
	token := &cobra.Command{Use: "token", Short: "Manage API tokens"}
	token.AddCommand(app.tokenCreateCommand(), app.tokenListCommand(), app.tokenRevokeCommand())
	return token
}

func (app *app) tokenCreateCommand() *cobra.Command {
	var body tokenCreateRequest
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a token; the secret is shown once",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if body.Name == "" || body.Scope == "" {
				return UsageError{errors.New("token create needs --name and --scope")}
			}
			return app.call(cmd, http.MethodPost, "/api/tokens", body)
		},
	}
	flags := create.Flags()
	flags.StringVar(&body.Name, "name", "", "token name, recorded as via token:<name>")
	flags.StringVar(&body.Scope, "scope", "", "read, write or admin")
	flags.StringArrayVar(&body.Repos, "repo", nil, "limit the token to this repository (repeatable)")
	flags.StringVar(&body.ExpiresIn, "expires", "", "lifetime as a Go duration, such as 720h")
	return create
}

func (app *app) tokenListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tokens",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, "/api/tokens", nil)
		},
	}
}

func (app *app) tokenRevokeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke a token",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodDelete, "/api/tokens/"+url.PathEscape(args[0]), nil)
		},
	}
}
