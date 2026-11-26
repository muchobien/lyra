package cli

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) repoCommand() *cobra.Command {
	repo := &cobra.Command{Use: "repo", Short: "Manage repositories"}
	repo.AddCommand(
		app.repoCreateCommand(),
		app.repoListCommand(),
		app.repoViewCommand(),
		app.repoDeleteCommand(),
	)
	repo.AddCommand(app.repoScoped(app.repoArchiveCommand())...)
	return repo
}

type downloadedArchive struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (app *app) repoArchiveCommand() *cobra.Command {
	var format, output string
	archive := &cobra.Command{
		Use:   "archive [<ref>]",
		Short: "Download the files at a ref as a .tar.gz or .zip archive",
		Long: `Download the files at a branch, tag or commit (default: the default branch), without
history, inside a <repo>-<ref> directory. The archive is saved as <repo>-<ref>.<format> in
the current directory, or where --output says; --output - writes it to stdout. Prints the
saved file's path and size.`,
		Args: MaximumArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "tar.gz" && format != "zip" {
				return UsageError{fmt.Errorf("--format is tar.gz or zip, not %q", format)}
			}
			owner, repo := splitRepo(args[0])
			ref := ""
			if len(args) > 1 {
				ref = args[1]
			} else {
				view, _, err := decode[lyraapi.RepoView](app.client.api.GetRepo(cmd.Context(), owner, repo))
				if err != nil {
					return err
				}
				ref = view.DefaultBranch
			}
			response, err := app.client.streaming.DownloadArchive(cmd.Context(), owner, repo, ref+"."+format)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			if response.StatusCode >= 400 {
				return responseError(response)
			}
			if output == "-" {
				_, err := io.Copy(cmd.OutOrStdout(), response.Body)
				return err
			}
			if output == "" {
				_, params, _ := mime.ParseMediaType(response.Header.Get("Content-Disposition"))
				output = filepath.Base(cmp.Or(params["filename"], repo+"."+format))
			}
			temporary, err := os.CreateTemp(filepath.Dir(output), ".lyra-download-*")
			if err != nil {
				return err
			}
			defer os.Remove(temporary.Name())
			size, err := io.Copy(temporary, response.Body)
			if closeErr := temporary.Close(); err == nil {
				err = closeErr
			}
			if err != nil {
				return err
			}
			if err := os.Rename(temporary.Name(), output); err != nil {
				return err
			}
			return app.emitIndented(cmd, downloadedArchive{Path: output, Size: size})
		},
	}
	archive.Flags().StringVar(&format, "format", "tar.gz", "tar.gz or zip")
	archive.Flags().StringVarP(&output, "output", "O", "", "file to save the archive to, or - for stdout")
	return archive
}

func (app *app) repoCreateCommand() *cobra.Command {
	var description string
	create := &cobra.Command{
		Use:   "create <owner>/<name>",
		Short: "Create a repository under an existing owner",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := lyraapi.CreateRepoRequest{Name: args[0], Description: &description}
			return app.show(cmd)(app.client.api.CreateRepo(cmd.Context(), body))
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
			return app.show(cmd)(app.client.api.ListRepos(cmd.Context()))
		},
	}
}

func (app *app) repoViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view [<owner>/<name>]",
		Short: "Show a repository with its branches and clone URL",
		Long: `Show a repository with its branches and clone URL. Without a name, it shows the
repository LYRA_REPO or the git remote points at.`,
		Args: MaximumArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := os.Getenv("LYRA_REPO")
			if len(args) > 0 {
				name = args[0]
			}
			if name == "" {
				resolved, err := app.client.remoteRepository(cmd)
				if err != nil {
					return err
				}
				name = resolved
			}
			owner, repo := splitRepo(name)
			return app.show(cmd)(app.client.api.GetRepo(cmd.Context(), owner, repo))
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
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.DeleteRepo(cmd.Context(), owner, repo))
		},
	}
	remove.Flags().BoolVarP(&yes, "yes", "y", false, "confirm permanent deletion")
	return remove
}
