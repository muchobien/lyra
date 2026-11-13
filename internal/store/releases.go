package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

var ErrExists = errors.New("already exists")

type Release struct {
	ID          int64      `json:"id"`
	Repo        string     `json:"repo"`
	Tag         string     `json:"tag"`
	Title       string     `json:"title"`
	Notes       string     `json:"notes"`
	Draft       bool       `json:"draft"`
	Prerelease  bool       `json:"prerelease"`
	Target      string     `json:"target"`
	Commit      string     `json:"commit"`
	Author      Actor      `json:"author"`
	Assets      []Asset    `json:"assets"`
	Images      []Image    `json:"images,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at"`
}

type Asset struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	ContentType string    `json:"content_type"`
	Uploader    Actor     `json:"uploader"`
	CreatedAt   time.Time `json:"created_at"`
	DownloadURL string    `json:"download_url,omitempty"`
	Object      string    `json:"-"`
}

func assetFromRow(row queries.ReleaseAsset) Asset {
	return Asset{
		ID: row.ID, Name: row.Name, Size: row.Size, SHA256: row.Sha256, ContentType: row.ContentType,
		Uploader:  Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		CreatedAt: row.CreatedAt, Object: row.Object,
	}
}

func loadRelease(q *queries.Queries, row queries.Release) (Release, error) {
	release := Release{
		ID: row.ID, Repo: row.Repo, Tag: row.Tag, Title: row.Title, Notes: row.Notes, Draft: row.Draft, Prerelease: row.Prerelease,
		Target: row.Target, Commit: row.CommitSha, Author: Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		Assets: []Asset{}, CreatedAt: row.CreatedAt,
	}
	if row.PublishedAt.Valid {
		release.PublishedAt = &row.PublishedAt.Time
	}
	rows, err := q.ListAssets(context.Background(), row.ID)
	if err != nil {
		return release, err
	}
	for _, asset := range rows {
		release.Assets = append(release.Assets, assetFromRow(asset))
	}
	return release, nil
}

func releaseEvent(release Release) map[string]any {
	names := []string{}
	for _, asset := range release.Assets {
		names = append(names, asset.Name)
	}
	return map[string]any{
		"id": release.ID, "tag": release.Tag, "title": release.Title, "draft": release.Draft, "prerelease": release.Prerelease,
		"commit": release.Commit, "assets": names,
	}
}

func publishedAt(release Release) sql.NullTime {
	if release.PublishedAt == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *release.PublishedAt, Valid: true}
}

func (s *Store) CreateRelease(actor Actor, release Release) (Release, int64, error) {
	var created Release
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		row, err := q.CreateRelease(context.Background(), queries.CreateReleaseParams{
			Repo: release.Repo, Tag: release.Tag, Title: release.Title, Notes: release.Notes, Draft: release.Draft, Prerelease: release.Prerelease,
			Target: release.Target, CommitSha: release.Commit, User: actor.User, Via: actor.Via, Agent: actor.Agent, Session: actor.Session,
			CreatedAt: now(), PublishedAt: publishedAt(release),
		})
		if IsUniqueViolation(err) {
			return ErrExists
		}
		if err != nil {
			return err
		}
		if created, err = loadRelease(q, row); err != nil {
			return err
		}
		action := "release.published"
		if created.Draft {
			action = "release.created"
		}
		eventID, err = addEvent(q, action, created.Repo, 0, actor, releaseEvent(created))
		return err
	})
	return created, eventID, err
}

func (s *Store) Release(repo string, id int64) (Release, error) {
	row, err := s.queries.GetRelease(context.Background(), queries.GetReleaseParams{Repo: repo, ID: id})
	if err != nil {
		return Release{}, notFound(err)
	}
	return loadRelease(s.queries, row)
}

func (s *Store) ReleaseByTag(repo, tag string) (Release, error) {
	row, err := s.queries.GetReleaseByTag(context.Background(), queries.GetReleaseByTagParams{Repo: repo, Tag: tag})
	if err != nil {
		return Release{}, notFound(err)
	}
	return loadRelease(s.queries, row)
}

func (s *Store) LatestRelease(repo string) (Release, error) {
	row, err := s.queries.LatestRelease(context.Background(), repo)
	if err != nil {
		return Release{}, notFound(err)
	}
	return loadRelease(s.queries, row)
}

func (s *Store) Releases(repo string) ([]Release, error) {
	rows, err := s.queries.ListReleases(context.Background(), repo)
	if err != nil {
		return nil, err
	}
	releases := []Release{}
	for _, row := range rows {
		release, err := loadRelease(s.queries, row)
		if err != nil {
			return nil, err
		}
		releases = append(releases, release)
	}
	return releases, nil
}

func (s *Store) UpdateRelease(actor Actor, release Release, action string) (Release, int64, error) {
	var updated Release
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		row, err := q.UpdateRelease(context.Background(), queries.UpdateReleaseParams{
			Title: release.Title, Notes: release.Notes, Draft: release.Draft, Prerelease: release.Prerelease, Target: release.Target,
			CommitSha: release.Commit, PublishedAt: publishedAt(release), ID: release.ID,
		})
		if err != nil {
			return notFound(err)
		}
		if updated, err = loadRelease(q, row); err != nil {
			return err
		}
		eventID, err = addEvent(q, action, updated.Repo, 0, actor, releaseEvent(updated))
		return err
	})
	return updated, eventID, err
}

func (s *Store) DeleteRelease(actor Actor, release Release) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		if err := deleted(q.DeleteRelease(context.Background(), release.ID)); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(q, "release.deleted", release.Repo, 0, actor, releaseEvent(release))
		return err
	})
	return eventID, err
}

func assetEvent(release Release, asset Asset, change string) map[string]any {
	return map[string]any{
		"release_id": release.ID, "tag": release.Tag, "name": asset.Name, "size": asset.Size, "sha256": asset.SHA256, "change": change,
	}
}

func (s *Store) AddAsset(actor Actor, release Release, asset Asset, clobber bool) (Asset, *Asset, int64, error) {
	var created Asset
	var replaced *Asset
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		existing, err := q.GetAsset(ctx, queries.GetAssetParams{ReleaseID: release.ID, Name: asset.Name})
		switch {
		case err == nil && !clobber:
			return ErrExists
		case err == nil:
			old := assetFromRow(existing)
			replaced = &old
			if err := deleted(q.DeleteAsset(ctx, existing.ID)); err != nil {
				return err
			}
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}
		row, err := q.CreateAsset(ctx, queries.CreateAssetParams{
			ReleaseID: release.ID, Name: asset.Name, Object: asset.Object, Size: asset.Size, Sha256: asset.SHA256, ContentType: asset.ContentType,
			User: actor.User, Via: actor.Via, Agent: actor.Agent, Session: actor.Session, CreatedAt: now(),
		})
		if err != nil {
			return notFound(err)
		}
		created = assetFromRow(row)
		eventID, err = addEvent(q, "release.asset", release.Repo, 0, actor, assetEvent(release, created, "uploaded"))
		return err
	})
	return created, replaced, eventID, err
}

func (s *Store) DeleteAsset(actor Actor, release Release, name string) (Asset, int64, error) {
	var asset Asset
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		row, err := q.GetAsset(ctx, queries.GetAssetParams{ReleaseID: release.ID, Name: name})
		if err != nil {
			return notFound(err)
		}
		asset = assetFromRow(row)
		if err := deleted(q.DeleteAsset(ctx, row.ID)); err != nil {
			return err
		}
		eventID, err = addEvent(q, "release.asset", release.Repo, 0, actor, assetEvent(release, asset, "deleted"))
		return err
	})
	return asset, eventID, err
}
