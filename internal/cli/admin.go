package cli

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/muchobien/lyra/internal/cli/lyraapi"
)

func (app *app) adminCommand() *cobra.Command {
	admin := &cobra.Command{Use: "admin", Short: "Backups, read-only mode and status (unrestricted admin)"}
	admin.AddCommand(
		app.adminBackupCommand(),
		app.adminReadOnlyCommand(),
		app.adminStatusCommand(),
		app.adminStorageCheckCommand(),
		app.adminRegistryGCCommand(),
	)
	return admin
}

func (app *app) adminBackupCommand() *cobra.Command {
	var output string
	backup := &cobra.Command{
		Use:   "backup",
		Short: "Write a consistent backup of the database and every repository as a tar file",
		Long: `Write a consistent backup as a tar file: lyra.db (made with VACUUM INTO), every bare
repository, and manifest.json (repository refs, and the stored files and images the
database refers to). Writes to Lyra pause while it's taken; the pause is reported.

Signing and secrets keys aren't included, nor are the files in object storage (release
assets, registry layers, job logs): back those up separately. Restore with lyra restore.
The command fails if the tar ends without its manifest, so a cut-off download never
passes as a backup.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.writeBackup(cmd, output)
		},
	}
	backup.Flags().StringVar(&output, "output", "", "file to write (default: stdout)")
	return backup
}

func (app *app) writeBackup(cmd *cobra.Command, output string) error {
	response, err := app.client.streaming.CreateBackup(cmd.Context())
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return responseError(response)
	}
	out := cmd.OutOrStdout()
	var file *os.File
	if output != "" && output != "-" {
		if file, err = os.Create(output); err != nil {
			return err
		}
		defer file.Close()
		out = file
	}
	if err := copyBackup(response.Body, out); err != nil {
		return err
	}
	if file != nil {
		if err := file.Sync(); err != nil {
			return err
		}
		pause := response.Header.Get("Lyra-Backup-Pause-Ms")
		fmt.Fprintf(cmd.ErrOrStderr(), "{\"file\":%q,\"pause_ms\":%s}\n", output, pause)
	}
	return nil
}

func copyBackup(body io.Reader, out io.Writer) error {
	archive := tar.NewReader(io.TeeReader(body, out))
	last := ""
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("backup stream: %w", err)
		}
		last = header.Name
	}
	if _, err := io.Copy(io.Discard, io.TeeReader(body, out)); err != nil {
		return err
	}
	if last != "manifest.json" {
		return errors.New("the backup ended without its manifest; it's incomplete")
	}
	return nil
}

func (app *app) adminReadOnlyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "readonly on|off|status",
		Short: "Refuse every write with 503 while reads keep working; a restart turns it off",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "on", "off":
				body := lyraapi.ReadOnlyRequest{Enabled: args[0] == "on"}
				return app.show(cmd)(app.client.api.SetReadOnly(cmd.Context(), body))
			case "status":
				return app.show(cmd)(app.client.api.GetStatus(cmd.Context()))
			}
			return UsageError{fmt.Errorf("readonly takes on, off or status, not %q", args[0])}
		},
	}
}

func (app *app) adminStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show read-only mode and the last backup",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.show(cmd)(app.client.api.GetStatus(cmd.Context()))
		},
	}
}

func (app *app) adminStorageCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "storage-check",
		Short: "List stored files and images the database refers to but storage lacks; exit 1 if any",
		Args:  ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.checkStorage(cmd)
		},
	}
}

func (app *app) checkStorage(cmd *cobra.Command) error {
	result, output, err := decode[lyraapi.StorageCheck](app.client.api.CheckStorage(cmd.Context()))
	if err != nil {
		return err
	}
	if _, err := cmd.OutOrStdout().Write(output); err != nil {
		return err
	}
	if len(result.Missing) > 0 {
		return fmt.Errorf("%d stored objects are missing", len(result.Missing))
	}
	return nil
}

func (app *app) adminRegistryGCCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "registry-gc",
		Short: "Delete registry layers and manifests no tag refers to",
		Long: `Delete registry layers and manifests that no tag refers to, directly or through a
multi-arch index, such as those left by lyra image delete, retagging or deleted repositories.
Lyra is read-only while it runs (pushes and other writes get 503), then returns to its
previous mode. Prints the blob count before and after, and the bytes freed.`,
		Args: ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.show(cmd)(app.client.api.CollectRegistry(cmd.Context()))
		},
	}
}
