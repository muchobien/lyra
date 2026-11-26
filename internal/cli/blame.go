package cli

import (
	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

type blameOutput struct {
	Path   string      `json:"path"`
	Binary bool        `json:"binary"`
	Ranges []blameText `json:"ranges"`
}

type blameText struct {
	Commit lyraapi.BlameCommit `json:"commit"`
	Start  int64               `json:"start"`
	Lines  []string            `json:"lines"`
}

func (app *app) blameCommand() *cobra.Command {
	var ref string
	blame := &cobra.Command{
		Use:   "blame <path>",
		Short: "Show the commit that last changed each line of a file",
		Long: `Show the commit that last changed each line of a file, as git blame would. Lines come in
ranges of consecutive lines from the same commit: its sha, author, date and summary, the
number of the range's first line, and the lines. Binary and large files have binary set
and no ranges.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			params := lyraapi.GetRepositoryBlameParams{Path: &args[1]}
			if ref != "" {
				params.Ref = &ref
			}
			blame, _, err := decode[lyraapi.BlameResponse](
				app.client.api.GetRepositoryBlame(cmd.Context(), owner, repo, &params),
			)
			if err != nil {
				return err
			}
			output := blameOutput{Path: blame.Path, Binary: blame.Binary, Ranges: []blameText{}}
			for _, entry := range blame.Ranges {
				text := blameText{Commit: entry.Commit, Start: entry.Start, Lines: entry.Text}
				output.Ranges = append(output.Ranges, text)
			}
			return app.emitIndented(cmd, output)
		},
	}
	blame.Flags().StringVar(&ref, "ref", "", "branch, tag or commit (default: the default branch)")
	return app.repoScoped(blame)[0]
}
