package backup

import (
	"archive/tar"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

const Format = 1

type Repo struct {
	Name string            `json:"name"`
	Refs map[string]string `json:"refs"`
}

type Manifest struct {
	Format    int                  `json:"format"`
	CreatedAt time.Time            `json:"created_at"`
	Schema    int64                `json:"schema"`
	Repos     []Repo               `json:"repos"`
	Objects   []store.StoredObject `json:"objects"`
}

type Report struct {
	Data    string `json:"data"`
	Schema  int64  `json:"schema"`
	Repos   int    `json:"repos"`
	Objects int    `json:"objects"`
}

func entryPath(name string) (string, error) {
	clean := path.Clean(name)
	if clean != strings.TrimSuffix(name, "/") || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe path %q in the backup", name)
	}
	if clean != "lyra.db" && clean != "manifest.json" && clean != "repos" && !strings.HasPrefix(clean, "repos/") {
		return "", fmt.Errorf("unexpected entry %q in the backup", name)
	}
	return clean, nil
}

func prepareTarget(dir string) error {
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) > 0 {
		return fmt.Errorf("%s isn't empty; restore into a new directory", dir)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.MkdirAll(dir, 0o750)
}

func writeFile(source io.Reader, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode&0o755|0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, source)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func extractEntry(archive *tar.Reader, header *tar.Header, name, dir string) (*Manifest, error) {
	target := filepath.Join(dir, filepath.FromSlash(name))
	switch header.Typeflag {
	case tar.TypeDir:
		return nil, os.MkdirAll(target, 0o750)
	case tar.TypeReg:
		if name != "manifest.json" {
			return nil, writeFile(archive, target, os.FileMode(header.Mode))
		}
		manifest := &Manifest{}
		if err := json.UnmarshalDecode(jsontext.NewDecoder(archive), manifest); err != nil {
			return nil, fmt.Errorf("manifest: %w", err)
		}
		return manifest, nil
	default:
		return nil, fmt.Errorf("unexpected entry type %q for %s", header.Typeflag, name)
	}
}

func extract(reader io.Reader, dir string) (*Manifest, error) {
	var manifest *Manifest
	archive := tar.NewReader(reader)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return manifest, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read backup: %w", err)
		}
		name, err := entryPath(header.Name)
		if err != nil {
			return nil, err
		}
		found, err := extractEntry(archive, header, name, dir)
		if err != nil {
			return nil, err
		}
		if found != nil {
			manifest = found
		}
	}
}

func checkDatabase(ctx context.Context, dir string) (int64, error) {
	db, err := store.Open(filepath.Join(dir, "lyra.db"))
	if err != nil {
		return 0, err
	}
	defer db.Close()
	if err := db.IntegrityCheck(ctx); err != nil {
		return 0, err
	}
	schema, err := db.SchemaVersion(ctx)
	if err != nil {
		return schema, fmt.Errorf("the backup has no usable lyra.db: %w", err)
	}
	latest, err := store.LatestMigration()
	if err != nil {
		return schema, err
	}
	if schema > latest {
		return schema, fmt.Errorf(
			"the backup's schema is %d, newer than this Lyra's %d; restore with a newer binary",
			schema,
			latest,
		)
	}
	return schema, nil
}

func checkRepos(ctx context.Context, dir string, expected []Repo) error {
	git := repos.Git{}
	for _, repo := range expected {
		repoDir := filepath.Join(dir, "repos", filepath.FromSlash(repo.Name)+".git")
		if err := git.Fsck(ctx, repoDir); err != nil {
			return fmt.Errorf("%s: %w", repo.Name, err)
		}
		refs, err := git.Refs(ctx, repoDir)
		if err != nil {
			return fmt.Errorf("%s: %w", repo.Name, err)
		}
		if !maps.Equal(refs, repo.Refs) {
			return fmt.Errorf("%s: refs differ from the manifest", repo.Name)
		}
	}
	return nil
}

func Restore(ctx context.Context, reader io.Reader, dir string) (Report, error) {
	report := Report{Data: dir}
	if err := prepareTarget(dir); err != nil {
		return report, err
	}
	manifest, err := extract(reader, dir)
	if err != nil {
		return report, err
	}
	if manifest == nil {
		return report, errors.New("the backup has no manifest.json; it's incomplete")
	}
	if manifest.Format != Format {
		return report, fmt.Errorf("backup format %d; this Lyra reads format %d", manifest.Format, Format)
	}
	if report.Schema, err = checkDatabase(ctx, dir); err != nil {
		return report, err
	}
	if err := checkRepos(ctx, dir, manifest.Repos); err != nil {
		return report, err
	}
	report.Repos, report.Objects = len(manifest.Repos), len(manifest.Objects)
	return report, nil
}
