package cli

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type releaseInfo struct {
	ID     int64          `json:"id"`
	Tag    string         `json:"tag"`
	Draft  bool           `json:"draft"`
	Assets []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type releaseCreateRequest struct {
	Tag           string `json:"tag"`
	Target        string `json:"target"`
	Title         string `json:"title"`
	Notes         string `json:"notes"`
	Draft         bool   `json:"draft"`
	Prerelease    bool   `json:"prerelease"`
	GenerateNotes bool   `json:"generate_notes"`
}

type releaseCreateOptions struct {
	target        string
	title         string
	draft         bool
	prerelease    bool
	generateNotes bool
	assets        []string
}

type releaseEditRequest struct {
	Title      *string `json:"title,omitzero"`
	Target     *string `json:"target,omitzero"`
	Prerelease *bool   `json:"prerelease,omitzero"`
	Notes      *string `json:"notes,omitzero"`
}

type releasePublishRequest struct {
	Draft bool `json:"draft"`
}

type downloadedAsset struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func releasesPath(repo string) string {
	return repoAPI(repo) + "/releases"
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

func (app *app) findRelease(cmd *cobra.Command, repo, tag string) (releaseInfo, string, error) {
	path := releasesPath(repo) + "/tags/" + strings.ReplaceAll(url.PathEscape(tag), "%2F", "/")
	if tag == "latest" {
		path = releasesPath(repo) + "/latest"
	}
	var found releaseInfo
	output, err := app.client.request(cmd.Context(), http.MethodGet, path, nil)
	if err != nil {
		return found, "", err
	}
	if err := json.Unmarshal(output, &found); err != nil {
		return found, "", err
	}
	return found, releasesPath(repo) + "/" + fmt.Sprint(found.ID), nil
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
	output, err := app.client.request(cmd.Context(), http.MethodPost, releasesPath(args[0]), releaseCreateRequest{
		Tag:           args[1],
		Target:        options.target,
		Title:         options.title,
		Notes:         notes,
		Draft:         options.draft || len(options.assets) > 0,
		Prerelease:    options.prerelease,
		GenerateNotes: options.generateNotes,
	})
	if err != nil {
		return err
	}
	if len(options.assets) == 0 {
		return app.emit(cmd, output)
	}
	var created releaseInfo
	if err := json.Unmarshal(output, &created); err != nil {
		return err
	}
	path := releasesPath(args[0]) + "/" + fmt.Sprint(created.ID)
	for _, file := range options.assets {
		if _, err := app.client.uploadAsset(cmd.Context(), path, file, false); err != nil {
			return fmt.Errorf("upload %s: %w; the release is still a draft", file, err)
		}
	}
	if options.draft {
		return app.call(cmd, http.MethodGet, path, nil)
	}
	return app.call(cmd, http.MethodPatch, path, releasePublishRequest{Draft: false})
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
			query := url.Values{"tag": {args[1]}, "target": {target}}
			if prerelease {
				query.Set("prerelease", "true")
			}
			return app.call(cmd, http.MethodGet, releasesPath(args[0])+"/notes?"+query.Encode(), nil)
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
			return app.call(cmd, http.MethodGet, releasesPath(args[0]), nil)
		},
	}
}

func (app *app) releaseViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view <repo> <tag|latest>",
		Short: "Show a release with its assets and download URLs",
		Args:  ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, path, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			return app.call(cmd, http.MethodGet, path, nil)
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
			var changes releaseEditRequest
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
			if changes == (releaseEditRequest{}) {
				flags := "--title, --notes, --notes-file, --prerelease or --target"
				return UsageError{errors.New("nothing to change; pass " + flags)}
			}
			_, path, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			return app.call(cmd, http.MethodPatch, path, changes)
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
			_, path, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			return app.call(cmd, http.MethodPatch, path, releasePublishRequest{Draft: false})
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
			_, path, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			if cleanupTag {
				path += "?cleanup_tag=true"
			}
			return app.call(cmd, http.MethodDelete, path, nil)
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
			_, path, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			uploaded := []jsontext.Value{}
			for _, file := range args[2:] {
				output, err := app.client.uploadAsset(cmd.Context(), path, file, clobber)
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
	found, path, err := app.findRelease(cmd, args[0], args[1])
	if err != nil {
		return err
	}
	files := []downloadedAsset{}
	for _, asset := range found.Assets {
		if matched, _ := filepath.Match(pattern, asset.Name); pattern != "" && !matched {
			continue
		}
		file := filepath.Join(dir, asset.Name)
		assetPath := path + "/assets/" + url.PathEscape(asset.Name)
		if err := app.client.downloadAsset(cmd.Context(), assetPath, file, asset.SHA256); err != nil {
			return fmt.Errorf("download %s: %w", asset.Name, err)
		}
		files = append(files, downloadedAsset{
			Name:   asset.Name,
			Path:   file,
			Size:   asset.Size,
			SHA256: asset.SHA256,
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
			_, path, err := app.findRelease(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			return app.call(cmd, http.MethodDelete, path+"/assets/"+url.PathEscape(args[2]), nil)
		},
	}
}
