package cli

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

func (app *app) commitCommand() *cobra.Command {
	commit := &cobra.Command{Use: "commit", Short: "Read commits and their signature status"}
	commit.AddCommand(app.commitListCommand(), app.commitShowCommand())
	return commit
}

func (app *app) commitListCommand() *cobra.Command {
	var ref string
	var limit int
	list := &cobra.Command{
		Use:   "list <repo>",
		Short: "List commits with signature status",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{"limit": {fmt.Sprint(limit)}}
			if ref != "" {
				query.Set("ref", ref)
			}
			return app.call(cmd, http.MethodGet, repoAPI(args[0])+"/commits?"+query.Encode(), nil)
		},
	}
	list.Flags().StringVar(&ref, "ref", "", "branch, tag or commit (default: the default branch)")
	list.Flags().IntVar(&limit, "limit", 30, "maximum number of commits")
	return list
}

func (app *app) commitShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <repo> <rev>",
		Short: "Show one commit, verified with git verify-commit",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.call(cmd, http.MethodGet, repoAPI(args[0])+"/commits/"+url.PathEscape(args[1]), nil)
		},
	}
}
