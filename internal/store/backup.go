package store

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/muchobien/lyra/internal/store/queries"
)

type StoredObject struct {
	Kind   string `json:"kind"`
	Repo   string `json:"repo"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Size   int64  `json:"size,omitzero"`
	SHA256 string `json:"sha256,omitempty"`
}

func (s *Store) Snapshot(ctx context.Context, file string) error {
	_, err := s.db.ExecContext(ctx, "VACUUM INTO ?", file)
	return err
}

func (s *Store) IntegrityCheck(ctx context.Context) error {
	var result string
	if err := s.db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("integrity check: %s", result)
	}
	return nil
}

func (s *Store) SchemaVersion(ctx context.Context) (int64, error) {
	var version int64
	query := "SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied"
	err := s.db.QueryRowContext(ctx, query).Scan(&version)
	return version, err
}

func LatestMigration() (int64, error) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return 0, err
	}
	var latest int64
	for _, entry := range entries {
		number, _, _ := strings.Cut(entry.Name(), "_")
		version, err := strconv.ParseInt(number, 10, 64)
		if err == nil && version > latest {
			latest = version
		}
	}
	return latest, nil
}

func (s *Store) StoredObjects(ctx context.Context) ([]StoredObject, error) {
	objects := []StoredObject{}
	assets, err := s.queries.ListAllAssets(ctx)
	if err != nil {
		return nil, err
	}
	for _, asset := range assets {
		objects = append(objects, StoredObject{
			Kind:   "release_asset",
			Repo:   asset.Repo,
			Name:   asset.Tag + "/" + asset.Name,
			Path:   "/lyra/releases/" + asset.Repo + "/" + asset.Object,
			Size:   asset.Size,
			SHA256: asset.Sha256,
		})
	}
	logs, err := s.queries.ListLogObjects(ctx)
	if err != nil {
		return nil, err
	}
	for _, log := range logs {
		objects = append(objects, StoredObject{
			Kind: "job_log",
			Repo: log.Repo,
			Name: "task " + strconv.FormatInt(log.ID, 10),
			Path: log.LogObject,
		})
	}
	images, err := s.queries.ListAllImages(ctx)
	if err != nil {
		return nil, err
	}
	for _, image := range images {
		algorithm, digest, _ := strings.Cut(image.Digest, ":")
		objects = append(objects, StoredObject{
			Kind: "image",
			Repo: image.Repo,
			Name: image.Name + ":" + image.Tag,
			Path: path.Join("/docker/registry/v2/blobs", algorithm, digest[:min(2, len(digest))], digest, "data"),
		})
	}
	return objects, nil
}

func (s *Store) LatestEvent(ctx context.Context, action string) (Event, error) {
	row, err := s.queries.LatestEventByAction(ctx, action)
	if err != nil {
		return Event{}, notFound(err)
	}
	return eventsFromRows([]queries.Event{row})[0], nil
}

type Counts struct {
	Repos int64
	Runs  map[string]int64
	Jobs  map[string]int64
}

func (s *Store) Counts(ctx context.Context) (Counts, error) {
	counts := Counts{Runs: map[string]int64{}, Jobs: map[string]int64{}}
	var err error
	if counts.Repos, err = s.queries.CountRepos(ctx); err != nil {
		return counts, err
	}
	runs, err := s.queries.CountRunsByStatus(ctx)
	if err != nil {
		return counts, err
	}
	for _, row := range runs {
		counts.Runs[row.Status] = row.Count
	}
	jobs, err := s.queries.CountJobsByStatus(ctx)
	if err != nil {
		return counts, err
	}
	for _, row := range jobs {
		counts.Jobs[row.Status] = row.Count
	}
	return counts, nil
}
