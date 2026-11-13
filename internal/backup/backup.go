package backup

import (
	"archive/tar"
	"encoding/json"
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

func Restore(reader io.Reader, dir string) (Report, error) {
	report := Report{Data: dir}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return report, fmt.Errorf("%s isn't empty; restore into a new directory", dir)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return report, err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return report, err
	}
	var manifest *Manifest
	archive := tar.NewReader(reader)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return report, fmt.Errorf("read backup: %w", err)
		}
		name, err := entryPath(header.Name)
		if err != nil {
			return report, err
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o750); err != nil {
				return report, err
			}
		case tar.TypeReg:
			if name == "manifest.json" {
				manifest = &Manifest{}
				if err := json.NewDecoder(archive).Decode(manifest); err != nil {
					return report, fmt.Errorf("manifest: %w", err)
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
				return report, err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(header.Mode)&0o755|0o600)
			if err != nil {
				return report, err
			}
			_, err = io.Copy(file, archive)
			if closeErr := file.Close(); err == nil {
				err = closeErr
			}
			if err != nil {
				return report, err
			}
		default:
			return report, fmt.Errorf("unexpected entry type %q for %s", header.Typeflag, name)
		}
	}
	if manifest == nil {
		return report, errors.New("the backup has no manifest.json; it's incomplete")
	}
	if manifest.Format != Format {
		return report, fmt.Errorf("backup format %d; this Lyra reads format %d", manifest.Format, Format)
	}
	db, err := store.Open(filepath.Join(dir, "lyra.db"))
	if err != nil {
		return report, err
	}
	defer db.Close()
	if err := db.IntegrityCheck(); err != nil {
		return report, err
	}
	if report.Schema, err = db.SchemaVersion(); err != nil {
		return report, fmt.Errorf("the backup has no usable lyra.db: %w", err)
	}
	latest, err := store.LatestMigration()
	if err != nil {
		return report, err
	}
	if report.Schema > latest {
		return report, fmt.Errorf("the backup's schema is %d, newer than this Lyra's %d; restore with a newer binary", report.Schema, latest)
	}
	git := repos.Git{}
	for _, repo := range manifest.Repos {
		repoDir := filepath.Join(dir, "repos", filepath.FromSlash(repo.Name)+".git")
		if err := git.Fsck(repoDir); err != nil {
			return report, fmt.Errorf("%s: %w", repo.Name, err)
		}
		refs, err := git.Refs(repoDir)
		if err != nil {
			return report, fmt.Errorf("%s: %w", repo.Name, err)
		}
		if !maps.Equal(refs, repo.Refs) {
			return report, fmt.Errorf("%s: refs differ from the manifest", repo.Name)
		}
	}
	report.Repos, report.Objects = len(manifest.Repos), len(manifest.Objects)
	return report, nil
}
