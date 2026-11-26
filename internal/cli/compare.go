package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) compareCommand() *cobra.Command {
	var diff bool
	compare := &cobra.Command{
		Use:   "compare [<base>...]<head>",
		Short: "Compare two refs: the commits head adds and the files it changes",
		Long: `Compare head with base, branches, tags or commits written as on the web, such as
main...feature/login. Without a base, head is compared with the default branch.

Prints the commits on head that aren't on base and the files head changed since the two
share history. --diff prints the unified diff instead.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			base, head, found := strings.Cut(args[1], "...")
			if !found {
				base, head = "", base
			}
			var baseParam *string
			if base != "" {
				baseParam = &base
			}
			if diff {
				params := lyraapi.GetCompareDiffParams{Base: baseParam, Head: head}
				return app.show(cmd)(app.client.api.GetCompareDiff(cmd.Context(), owner, repo, &params))
			}
			params := lyraapi.CompareRefsParams{Base: baseParam, Head: head}
			return app.show(cmd)(app.client.api.CompareRefs(cmd.Context(), owner, repo, &params))
		},
	}
	compare.Flags().BoolVar(&diff, "diff", false, "print the unified diff instead of JSON")
	return app.repoScoped(compare)[0]
}
