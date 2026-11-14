package cli

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

type releaseCreateOptions struct {
	target        string
	title         string
	draft         bool
	prerelease    bool
	generateNotes bool
	assets        []string
}

type downloadedAsset struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func (app *app) releaseCommand() *cobra.Command {
	release := &cobra.Command{Use: "release", Short: "Manage releases and their assets"}
	release.AddCommand(
		app.releaseCreateCommand(),
		app.releaseNotesCommand(),
		app.releaseListCommand(),
		app.releaseViewCommand(),
		app.releaseEditCommand(),
		app.releasePublishCommand(),
		app.releaseDeleteCommand(),
		app.releaseUploadCommand(),
		app.releaseDownloadCommand(),
		app.releaseDeleteAssetCommand(),
	)
	return release
}

func (app *app) findRelease(cmd *cobra.Command, repo, tag string) (lyraapi.Release, error) {
	owner, name := splitRepo(repo)
	if tag == "latest" {
		found, _, err := decode[lyraapi.Release](app.client.api.GetLatestRelease(cmd.Context(), owner, name))
		return found, err
	}
	found, _, err := decode[lyraapi.Release](app.client.api.GetReleaseByTag(cmd.Context(), owner, name, tag))
	return found, err
}

func (app *app) releaseCreateCommand() *cobra.Command {
	var options releaseCreateOptions
	create := &cobra.Command{
		Use:   "create <repo> <tag>",
		Short: "Create a release, uploading its assets before it's published",
		Long: `Create a release for a tag. If the tag doesn't exist, publishing creates it at --target
(default: the default branch). A draft doesn't create the tag until it's published.

With --asset, the release is created as a draft, the assets are uploaded, and then it is
published, so release.published fires once with every asset present. If an upload fails,
the release stays a draft; finish it with lyra release upload and lyra release publish.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.createRelease(cmd, args, options)
		},
	}
	flags := create.Flags()
	flags.StringVar(
		&options.target,
		"target",
		"",
		"branch or commit to tag if the tag doesn't exist (default: the default branch)",
	)
	flags.StringVar(&options.title, "title", "", "release title (default: the tag)")
	addTextFlags(create, "notes", "release notes in Markdown")
	flags.BoolVar(&options.draft, "draft", false, "keep the release as a draft")
	flags.BoolVar(
		&options.generateNotes,
		"generate-notes",
		false,
		"append a list of the pull requests merged since the previous release",
	)
	flags.BoolVar(
		&options.prerelease,
		"prerelease",
		false,
		"mark as a prerelease, so it's never the latest release",
	)
	flags.StringArrayVar(&options.assets, "asset", nil, "file to upload as an asset (repeatable)")
	return create
}

func (app *app) createRelease(cmd *cobra.Command, args []string, options releaseCreateOptions) error {
	notes, _, err := readText(cmd, "notes")
	if err != nil {
		return err
	}
	for _, file := range options.assets {
		if _, err := os.Stat(file); err != nil {
			return UsageError{err}
		}
	}
	draft := options.draft || len(options.assets) > 0
	body := lyraapi.CreateReleaseRequest{
		Tag:           args[1],
		Target:        &options.target,
		Title:         &options.title,
		Notes:         &notes,
		Draft:         &draft,
		Prerelease:    &options.prerelease,
		GenerateNotes: &options.generateNotes,
	}
	owner, repo := splitRepo(args[0])
	created, output, err := decode[lyraapi.Release](app.client.api.CreateRelease(cmd.Context(), owner, repo, body))
	if err != nil {
		return err
	}
	if len(options.assets) == 0 {
		return app.emit(cmd, output)
	}
	for _, file := range options.assets {
		if _, err := app.client.uploadAsset(cmd.Context(), args[0], created.Id, file, false); err != nil {
			return fmt.Errorf("upload %s: %w; the release is still a draft", file, err)
		}
	}
	if options.draft {
		return app.show(cmd)(app.client.api.GetRelease(cmd.Context(), owner, repo, created.Id))
	}
	publish := lyraapi.EditReleaseRequest{Draft: new(false)}
	return app.show(cmd)(app.client.api.EditRelease(cmd.Context(), owner, repo, created.Id, publish))
}

func (app *app) releaseNotesCommand() *cobra.Command {
	var target string
	var prerelease bool
	notes := &cobra.Command{
		Use:   "notes <repo> <tag>",
		Short: "Preview the notes --generate-notes would write",
		Long: `Preview the notes --generate-notes would write: the pull requests whose merge commit
is reachable from the tag (or --target, if the tag doesn't exist yet) but not from the
previous release. The previous release is the most recently published one whose commit
is an ancestor; prereleases count only with --prerelease.`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			params := lyraapi.PreviewReleaseNotesParams{Tag: &args[1], Target: &target, Prerelease: &prerelease}
			return app.show(cmd)(app.client.api.PreviewReleaseNotes(cmd.Context(), owner, repo, &params))
		},
	}
	flags := notes.Flags()
	flags.StringVar(
		&target,
		"target",
		"",
		"branch or commit, if the tag doesn't exist (default: the default branch)",
	)
	flags.BoolVar(
		&prerelease,
		"prerelease",
		false,
		"for a prerelease, so earlier prereleases count as the previous release",
	)
	return notes
}

func (app *app) releaseListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list <repo>",
		Short: "List releases, drafts first (drafts need write access)",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.ListReleases(cmd.Context(), owner, repo))
		},
	}
}

func (app *app) releaseViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view <repo> <tag|latest>",
		Short: "Show a release with its assets and download URLs",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			found, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.GetRelease(cmd.Context(), owner, repo, found.Id))
		},
	}
}

func (app *app) releaseEditCommand() *cobra.Command {
	var title, target string
	var prerelease bool
	edit := &cobra.Command{
		Use:   "edit <repo> <tag>",
		Short: "Change a release's title, notes or prerelease flag, or a draft's target",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var changes lyraapi.EditReleaseRequest
			if cmd.Flags().Changed("title") {
				changes.Title = &title
			}
			if cmd.Flags().Changed("target") {
				changes.Target = &target
			}
			if cmd.Flags().Changed("prerelease") {
				changes.Prerelease = &prerelease
			}
			notes, set, err := readText(cmd, "notes")
			if err != nil {
				return err
			}
			if set {
				changes.Notes = &notes
			}
			if changes == (lyraapi.EditReleaseRequest{}) {
				flags := "--title, --notes, --notes-file, --prerelease or --target"
				return UsageError{errors.New("nothing to change; pass " + flags)}
			}
			found, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.EditRelease(cmd.Context(), owner, repo, found.Id, changes))
		},
	}
	flags := edit.Flags()
	flags.StringVar(&title, "title", "", "release title")
	flags.StringVar(&target, "target", "", "branch or commit a draft's tag is created at")
	flags.BoolVar(&prerelease, "prerelease", false, "mark as a prerelease (--prerelease=false to clear)")
	addTextFlags(edit, "notes", "release notes in Markdown")
	return edit
}

func (app *app) releasePublishCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "publish <repo> <tag>",
		Short: "Publish a draft, creating its tag if needed",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			found, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			publish := lyraapi.EditReleaseRequest{Draft: new(false)}
			return app.show(cmd)(app.client.api.EditRelease(cmd.Context(), owner, repo, found.Id, publish))
		},
	}
}

func (app *app) releaseDeleteCommand() *cobra.Command {
	var cleanupTag, yes bool
	remove := &cobra.Command{
		Use:   "delete <repo> <tag>",
		Short: "Delete a release and its assets; the tag stays unless --cleanup-tag",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				message := "release delete removes its assets permanently; pass --yes"
				return UsageError{errors.New(message)}
			}
			found, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			params := lyraapi.DeleteReleaseParams{CleanupTag: &cleanupTag}
			return app.show(cmd)(app.client.api.DeleteRelease(cmd.Context(), owner, repo, found.Id, &params))
		},
	}
	remove.Flags().BoolVar(&cleanupTag, "cleanup-tag", false, "delete the tag too")
	remove.Flags().BoolVar(&yes, "yes", false, "confirm permanent deletion")
	return remove
}

func (app *app) releaseUploadCommand() *cobra.Command {
	var clobber bool
	upload := &cobra.Command{
		Use:   "upload <repo> <tag> <file>...",
		Short: "Upload files as assets of a release, named after the file",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				path := cmd.CommandPath()
				message := "%s takes a repo, a tag and at least one file; see %s --help"
				return UsageError{fmt.Errorf(message, path, path)}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			found, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			uploaded := []jsontext.Value{}
			for _, file := range args[2:] {
				output, err := app.client.uploadAsset(cmd.Context(), args[0], found.Id, file, clobber)
				if err != nil {
					return fmt.Errorf("upload %s: %w", file, err)
				}
				uploaded = append(uploaded, output)
			}
			return app.emitIndented(cmd, uploaded)
		},
	}
	upload.Flags().BoolVar(&clobber, "clobber", false, "replace assets that already exist")
	return upload
}

func (app *app) releaseDownloadCommand() *cobra.Command {
	var pattern, dir string
	download := &cobra.Command{
		Use:   "download <repo> <tag|latest>",
		Short: "Download a release's assets and check their sha256",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.downloadRelease(cmd, args, pattern, dir)
		},
	}
	download.Flags().StringVar(&pattern, "pattern", "", "only assets whose name matches this glob")
	download.Flags().StringVar(&dir, "dir", ".", "directory to write the files to")
	return download
}

func (app *app) downloadRelease(cmd *cobra.Command, args []string, pattern, dir string) error {
	if _, err := filepath.Match(pattern, ""); err != nil {
		return UsageError{fmt.Errorf("--pattern: %w", err)}
	}
	found, err := app.findRelease(cmd, args[0], args[1])
	if err != nil {
		return err
	}
	files := []downloadedAsset{}
	for _, asset := range found.Assets {
		if matched, _ := filepath.Match(pattern, asset.Name); pattern != "" && !matched {
			continue
		}
		file := filepath.Join(dir, asset.Name)
		if err := app.client.downloadAsset(cmd.Context(), args[0], found.Id, asset, file); err != nil {
			return fmt.Errorf("download %s: %w", asset.Name, err)
		}
		files = append(files, downloadedAsset{
			Name:   asset.Name,
			Path:   file,
			Size:   asset.Size,
			SHA256: asset.Sha256,
		})
	}
	if pattern != "" && len(files) == 0 {
		return fmt.Errorf("no asset of %s matches %s", found.Tag, pattern)
	}
	return app.emitIndented(cmd, files)
}

func (app *app) releaseDeleteAssetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-asset <repo> <tag> <name>",
		Short: "Delete one asset of a release",
		Args:  ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			found, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			owner, repo := splitRepo(args[0])
			return app.show(cmd)(app.client.api.DeleteAsset(cmd.Context(), owner, repo, found.Id, args[2]))
		},
	}
}
