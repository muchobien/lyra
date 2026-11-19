package cli

import (
	"github.com/spf13/cobra"
)

type app struct {
	client *client
	jq     string
	fields []string
}

func Root() *cobra.Command {
	app := &app{client: newClient()}
	root := &cobra.Command{
		Use:   "lyra",
		Short: "Client for the Lyra forge",
		Long: `Client for the Lyra forge. Output is JSON on stdout; errors are {"error": "..."} on stderr.
--json picks fields from it and --jq filters it, --jq last.

Environment:
  LYRA_URL      server URL (required)
  LYRA_TOKEN    API token; not needed on the tailnet behind tailscale-auth
  LYRA_AGENT    agent name recorded on writes (Lyra-Agent header); defaults to claude
                in Claude Code and codex in Codex
  LYRA_SESSION  session id recorded on writes (Lyra-Session header); defaults to the
                Claude Code or Codex session id
  LYRA_REPO     repository as owner/name for commands that act on one

Config file: $XDG_CONFIG_HOME/lyra/config.json (~/.config/lyra/config.json) sets
"url" and "token" when LYRA_URL and LYRA_TOKEN are unset, such as
{"url": "https://lyra.example.com", "token": "..."}. A file with a token must be
readable only by its owner (chmod 600).

Repositories are named <owner>/<name>, such as krosf/lyra; create the owner first with
lyra owner create. Commands that act on a repository take it from --repo (-R), then
LYRA_REPO, then the git remote of the working directory that points at the Lyra server
(origin when several point at different repositories).`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().StringVarP(&app.jq, "jq", "q", "", "filter JSON output with a jq expression")
	root.PersistentFlags().StringSliceVar(
		&app.fields,
		"json",
		nil,
		"keep only these comma-separated fields of the JSON output, or of each element of a list",
	)
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

func (app *app) meCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated actor and scope",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.show(cmd)(app.client.api.GetMe(cmd.Context()))
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
			return app.show(cmd)(app.client.api.GetSigningKey(cmd.Context()))
		},
	}
}
