package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) tokenCommand() *cobra.Command {
	token := &cobra.Command{Use: "token", Short: "Manage API tokens"}
	token.AddCommand(app.tokenCreateCommand(), app.tokenListCommand(), app.tokenRevokeCommand())
	return token
}

func (app *app) tokenCreateCommand() *cobra.Command {
	var name, scope, expires string
	var repos []string
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a token; the secret is shown once",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || scope == "" {
				return UsageError{errors.New("token create needs --name and --scope")}
			}
			body := lyraapi.CreateTokenRequest{
				Name:      name,
				Scope:     lyraapi.CreateTokenRequestScope(scope),
				Repos:     &repos,
				ExpiresIn: &expires,
			}
			return app.show(cmd)(app.client.api.CreateToken(cmd.Context(), body))
		},
	}
	flags := create.Flags()
	flags.StringVar(&name, "name", "", "token name, recorded as via token:<name>")
	flags.StringVar(&scope, "scope", "", "read, write or admin")
	flags.StringArrayVar(&repos, "repo", nil, "limit the token to this repository (repeatable)")
	flags.StringVar(&expires, "expires", "", "lifetime as a Go duration, such as 720h")
	return create
}

func (app *app) tokenListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tokens",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.show(cmd)(app.client.api.ListTokens(cmd.Context()))
		},
	}
}

func (app *app) tokenRevokeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke a token",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseNumber(args[0])
			if err != nil {
				return err
			}
			return app.show(cmd)(app.client.api.RevokeToken(cmd.Context(), id))
		},
	}
}
