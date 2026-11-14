package cli

import (
	"github.com/spf13/cobra"
)

func (app *app) tagCommand() *cobra.Command {
	tag := &cobra.Command{Use: "tag", Short: "Read tags and their signature status"}
	tag.AddCommand(app.tagListCommand())
	return tag
}

func (app *app) tagListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo>",
		Short: "List tags, verified with git verify-tag",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListTags(cmd.Context(), owner, repo))
		},
	}
}
