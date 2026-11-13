package cli

import (
	"net/http"

	"github.com/spf13/cobra"
)

type app struct {
	client *client
	jq     string
}

func Root() *cobra.Command {
	app := &app{client: newClient()}
	root := &cobra.Command{
		Use:   "lyra",
		Short: "Client for the Lyra forge",
		Long: `Client for the Lyra forge. Output is JSON on stdout; errors are {"error": "..."} on stderr.

Environment:
  LYRA_URL      server URL (default http://localhost:8080)
  LYRA_TOKEN    API token; not needed on the tailnet behind tailscale-auth
  LYRA_AGENT    agent name recorded on writes (Lyra-Agent header)
  LYRA_SESSION  session id recorded on writes (Lyra-Session header)

Repositories are named <owner>/<name>, such as krosf/lyra; create the owner first with
lyra owner create.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().StringVar(&app.jq, "jq", "", "filter JSON output with a jq expression")
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return UsageError{err}
	})
	root.AddCommand(
		app.signingKeyCommand(),
		app.meCommand(),
		app.repoCommand(),
		app.commitCommand(),
		app.tagCommand(),
		app.tokenCommand(),
		app.keyCommand(),
		app.eventCommand(),
		app.issueCommand(),
		app.prCommand(),
		app.labelCommand(),
		app.ruleCommand(),
		app.releaseCommand(),
		app.imageCommand(),
		app.runnerCommand(),
		app.ownerCommand(),
		app.runCommand(),
		app.workflowCommand(),
		app.secretCommand(),
		app.variableCommand(),
		app.adminCommand(),
	)
	return root
}

func (app *app) call(cmd *cobra.Command, method, path string, body any) error {
	output, err := app.client.request(cmd.Context(), method, path, body)
	if err != nil {
		return err
	}
	return app.emit(cmd, output)
}

func (app *app) meCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated actor and scope",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, "/api/me", nil)
		},
	}
}

func (app *app) signingKeyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "signing-key",
		Short: "Show the public key Lyra signs merge commits with",
		Long: "Show the public key Lyra signs merge commits with, " +
			"and an allowed_signers line for git verify-commit.",
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, "/api/signing-key", nil)
		},
	}
}
