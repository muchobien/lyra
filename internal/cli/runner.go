package cli

import (
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

type runnerCreateRequest struct {
	Name string `json:"name"`
}

type runnerTokenCreateRequest struct {
	Name string `json:"name"`
}

func (app *app) runnerCommand() *cobra.Command {
	runner := &cobra.Command{Use: "runner", Short: "Manage Actions runners and their registration tokens"}
	runner.AddCommand(
		app.runnerCreateCommand(),
		app.runnerListCommand(),
		app.runnerDeleteCommand(),
		app.runnerTokenCommand(),
	)
	return runner
}

func (app *app) runnerListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered runners, with labels, version and whether they polled in the last minute",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, "/api/runners", nil)
		},
	}
}

func (app *app) runnerCreateCommand() *cobra.Command {
	var body runnerCreateRequest
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a runner and print its uuid and token, shown once",
		Long: `Create a runner and print its uuid and token, shown once, for a connection in the
forgejo-runner config file instead of forgejo-runner register:

  server:
    connections:
      lyra:
        url: <url>
        uuid: <uuid>
        token: <token>
        labels: [...]

The runner declares its labels and version when it starts. Many processes can share one
runner, for example forgejo-runner one-job in a fresh pod per job.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodPost, "/api/runners", body)
		},
	}
	create.Flags().StringVar(&body.Name, "name", "", "runner name (required)")
	return create
}

func (app *app) runnerDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a runner; it can't fetch jobs any more",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodDelete, "/api/runners/"+url.PathEscape(args[0]), nil)
		},
	}
}

func (app *app) runnerTokenCommand() *cobra.Command {
	token := &cobra.Command{Use: "token", Short: "Manage runner registration tokens"}
	token.AddCommand(
		app.runnerTokenCreateCommand(),
		app.runnerTokenListCommand(),
		app.runnerTokenDeleteCommand(),
	)
	return token
}

func (app *app) runnerTokenCreateCommand() *cobra.Command {
	var body runnerTokenCreateRequest
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a registration token, shown once",
		Long: `Create a registration token, shown once in "secret". Register a runner with it:

  forgejo-runner register --no-interactive --instance <url> --token <secret> --name <name> --labels <labels>

"url" is the address runners reach Lyra at (--actions-url on the server). A token can
register any number of runners, such as one ephemeral runner per job, until it's deleted.
Deleting it doesn't remove the runners it registered.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodPost, "/api/runners/tokens", body)
		},
	}
	create.Flags().StringVar(&body.Name, "name", "", "what the token is for (required)")
	return create
}

func (app *app) runnerTokenListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registration tokens",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, "/api/runners/tokens", nil)
		},
	}
}

func (app *app) runnerTokenDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a registration token",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodDelete, "/api/runners/tokens/"+url.PathEscape(args[0]), nil)
		},
	}
}
