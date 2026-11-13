package cli

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

type keyAddRequest struct {
	Signing   bool   `json:"signing"`
	Agent     string `json:"agent"`
	Title     string `json:"title"`
	PublicKey string `json:"public_key"`
}

func (app *app) keyCommand() *cobra.Command {
	key := &cobra.Command{Use: "key", Short: "Manage SSH signing keys"}
	key.AddCommand(app.keyAddCommand(), app.keyListCommand(), app.keyDeleteCommand())
	return key
}

func (app *app) keyAddCommand() *cobra.Command {
	var body keyAddRequest
	add := &cobra.Command{
		Use:   "add <public-key-file|->",
		Short: "Register an SSH public key for commit and tag signatures",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !body.Signing {
				return UsageError{errors.New("only signing keys are supported; pass --signing")}
			}
			publicKey, err := readInput(cmd, args[0])
			if err != nil {
				return err
			}
			if bytes.Contains(publicKey, []byte("PRIVATE KEY")) {
				return UsageError{errors.New("that is a private key; pass the .pub file")}
			}
			body.PublicKey = strings.TrimSpace(string(publicKey))
			return app.call(cmd, http.MethodPost, "/api/keys", body)
		},
	}
	add.Flags().BoolVar(&body.Signing, "signing", false, "register as a signing key (required)")
	add.Flags().StringVar(&body.Agent, "agent", "", "label the key as belonging to this agent")
	add.Flags().StringVar(&body.Title, "title", "", "key title (default: the key comment)")
	return add
}

func (app *app) keyListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List keys",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, "/api/keys", nil)
		},
	}
}

func (app *app) keyDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a key",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodDelete, "/api/keys/"+url.PathEscape(args[0]), nil)
		},
	}
}
