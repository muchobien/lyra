package cli

import (
	"bytes"
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) keyCommand() *cobra.Command {
	key := &cobra.Command{Use: "key", Short: "Manage SSH signing keys"}
	key.AddCommand(app.keyAddCommand(), app.keyListCommand(), app.keyDeleteCommand())
	return key
}

func (app *app) keyAddCommand() *cobra.Command {
	var signing bool
	var agent, title string
	add := &cobra.Command{
		Use:   "add <public-key-file|->",
		Short: "Register an SSH public key for commit and tag signatures",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !signing {
				return UsageError{errors.New("only signing keys are supported; pass --signing")}
			}
			publicKey, err := readInput(cmd, args[0])
			if err != nil {
				return err
			}
			if bytes.Contains(publicKey, []byte("PRIVATE KEY")) {
				return UsageError{errors.New("that is a private key; pass the .pub file")}
			}
			body := lyraapi.CreateKeyRequest{
				Signing:   true,
				Agent:     &agent,
				Title:     &title,
				PublicKey: strings.TrimSpace(string(publicKey)),
			}
			return app.show(cmd)(app.client.api.CreateKey(cmd.Context(), body))
		},
	}
	add.Flags().BoolVar(&signing, "signing", false, "register as a signing key (required)")
	add.Flags().StringVar(&agent, "agent", "", "label the key as belonging to this agent")
	add.Flags().StringVar(&title, "title", "", "key title (default: the key comment)")
	return add
}

func (app *app) keyListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List keys",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.show(cmd)(app.client.api.ListKeys(cmd.Context()))
		},
	}
}

func (app *app) keyDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a key",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseNumber(args[0])
			if err != nil {
				return err
			}
			return app.show(cmd)(app.client.api.DeleteKey(cmd.Context(), id))
		},
	}
}
