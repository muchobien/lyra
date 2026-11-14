package cli

import (
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) secretCommand() *cobra.Command {
	secret := &cobra.Command{Use: "secret", Short: "Manage Actions secrets of a repository (admin)"}
	secret.AddCommand(app.secretSetCommand(), app.secretListCommand(), app.secretDeleteCommand())
	return secret
}

func (app *app) secretSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <repo> <NAME>",
		Short: "Set a secret from stdin; one trailing newline is dropped",
		Long: `Set a secret from stdin, so the value stays out of shell history: for example
lyra secret set krosf/lyra DEPLOY_KEY < key.txt. One trailing newline is dropped. Names are
upper-cased. Jobs read it as ${{ secrets.NAME }}, and runners mask it in logs.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			body := lyraapi.ValueRequest{Value: strings.TrimSuffix(string(value), "\n")}
			return app.show(cmd)(app.client.api.SetSecret(cmd.Context(), owner, repo, args[1], body))
		},
	}
}

func (app *app) secretListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo>",
		Short: "List secret names, never their values",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListSecrets(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) secretDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <repo> <NAME>",
		Short: "Delete a secret",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.DeleteSecret(cmd.Context(), owner, repo, args[1]))
		},
	}
}
