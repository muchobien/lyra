package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func (app *app) imageCommand() *cobra.Command {
	image := &cobra.Command{Use: "image", Short: "List and delete container images"}
	image.AddCommand(app.imageListCommand(), app.imageDeleteCommand())
	return image
}

func (app *app) imageListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo>",
		Short: "List a repository's image tags with their digests and pull references",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListImages(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) imageDeleteCommand() *cobra.Command {
	var yes bool
	remove := &cobra.Command{
		Use:   "delete <name>:<tag>|<name>@<digest>",
		Short: "Delete an image tag, or a manifest and every tag pointing at it",
		Long: `Delete an image tag, such as lyra/runner:v1, or a manifest by digest, such as
lyra@sha256:…, which also removes every tag pointing at it. Needs admin access. Layers stay
in storage until lyra admin registry-gc frees them.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return UsageError{errors.New("image delete can't be undone; pass --yes")}
			}
			name, reference, found := strings.Cut(args[0], "@")
			if !found {
				colon := strings.LastIndex(args[0], ":")
				if colon < 0 || colon < strings.LastIndex(args[0], "/") {
					return UsageError{fmt.Errorf("%s has no tag or digest", args[0])}
				}
				name, reference = args[0][:colon], args[0][colon+1:]
			}
			api := app.client.api
			path := api.Server + "v2/" + name + "/manifests/" + url.PathEscape(reference)
			req, err := http.NewRequestWithContext(cmd.Context(), http.MethodDelete, path, nil)
			if err != nil {
				return err
			}
			for _, edit := range api.RequestEditors {
				if err := edit(cmd.Context(), req); err != nil {
					return err
				}
			}
			if _, err := read(api.Client.Do(req)); err != nil {
				return err
			}
			return app.emit(cmd, []byte(`{"deleted": `+strconv.Quote(args[0])+"}\n"))
		},
	}
	remove.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return remove
}
