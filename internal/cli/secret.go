package cli

import (
	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) secretCommand() *cobra.Command {
	secret := &cobra.Command{Use: "secret", Short: "Manage Actions secrets of a repository (admin)"}
	secret.AddCommand(app.repoScoped(app.secretSetCommand(), app.secretListCommand(), app.secretDeleteCommand())...)
	return secret
}

func (app *app) secretSetCommand() *cobra.Command {
	set := &cobra.Command{
		Use:   "set <NAME>",
		Short: "Set a secret from --body or stdin",
		Long: `Set a secret from --body, or from stdin so the value stays out of shell history: for
example lyra secret set DEPLOY_KEY < key.txt. One trailing newline is dropped from stdin.
Names are upper-cased. Jobs read it as ${{ secrets.NAME }}, and runners mask it in logs.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := readValue(cmd)
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			body := lyraapi.ValueRequest{Value: value}
			return app.show(cmd)(app.client.api.SetSecret(cmd.Context(), owner, repo, args[1], body))
		},
	}
	set.Flags().StringP("body", "b", "", "the value (default: read from stdin)")
	return set
}

func (app *app) secretListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List secret names, never their values",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListSecrets(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) secretDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <NAME>",
		Short: "Delete a secret",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.DeleteSecret(cmd.Context(), owner, repo, args[1]))
		},
	}
}
