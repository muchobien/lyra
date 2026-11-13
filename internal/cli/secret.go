package cli

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

type secretSetRequest struct {
	Value string `json:"value"`
}

func secretPath(repo, name string) string {
	return repoAPI(repo) + "/secrets/" + url.PathEscape(name)
}

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
			body := secretSetRequest{Value: strings.TrimSuffix(string(value), "\n")}
			return app.call(cmd, http.MethodPut, secretPath(args[0], args[1]), body)
		},
	}
}

func (app *app) secretListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo>",
		Short: "List secret names, never their values",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, repoAPI(args[0])+"/secrets", nil)
		},
	}
}

func (app *app) secretDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <repo> <NAME>",
		Short: "Delete a secret",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodDelete, secretPath(args[0], args[1]), nil)
		},
	}
}
