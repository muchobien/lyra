package cli

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

type repoCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func repoAPI(name string) string {
	owner, repo, _ := strings.Cut(name, "/")
	return "/api/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
}

func (app *app) repoCommand() *cobra.Command {
	repo := &cobra.Command{Use: "repo", Short: "Manage repositories"}
	repo.AddCommand(
		app.repoCreateCommand(),
		app.repoListCommand(),
		app.repoGetCommand(),
		app.repoDeleteCommand(),
	)
	return repo
}

func (app *app) repoCreateCommand() *cobra.Command {
	var description string
	create := &cobra.Command{
		Use:   "create <owner>/<name>",
		Short: "Create a repository under an existing owner",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := repoCreateRequest{Name: args[0], Description: description}
			return app.call(cmd, http.MethodPost, "/api/repos", body)
		},
	}
	create.Flags().StringVar(&description, "description", "", "repository description")
	return create
}

func (app *app) repoListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, "/api/repos", nil)
		},
	}
}

func (app *app) repoGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Show a repository with its branches and clone URL",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, repoAPI(args[0]), nil)
		},
	}
}

func (app *app) repoDeleteCommand() *cobra.Command {
	var yes bool
	remove := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a repository permanently",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				message := "repo delete removes the repository permanently; pass --yes"
				return UsageError{errors.New(message)}
			}
			return app.call(cmd, http.MethodDelete, repoAPI(args[0]), nil)
		},
	}
	remove.Flags().BoolVar(&yes, "yes", false, "confirm permanent deletion")
	return remove
}
