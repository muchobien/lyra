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

type releaseEvent struct {
	Assets     []string `json:"assets"`
	Commit     string   `json:"commit"`
	Draft      bool     `json:"draft"`
	ID         int64    `json:"id"`
	Prerelease bool     `json:"prerelease"`
	Tag        string   `json:"tag"`
	Title      string   `json:"title"`
}

type assetEvent struct {
	Change    string `json:"change"`
	Name      string `json:"name"`
	ReleaseID int64  `json:"release_id"`
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	Tag       string `json:"tag"`
}

func assetFromRow(row queries.ReleaseAsset) Asset {
	return Asset{
		ID:          row.ID,
		Name:        row.Name,
		Size:        row.Size,
		SHA256:      row.Sha256,
		ContentType: row.ContentType,
		Uploader:    Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		CreatedAt:   row.CreatedAt,
		Object:      row.Object,
	}
}

func loadRelease(ctx context.Context, q *queries.Queries, row queries.Release) (Release, error) {
	release := Release{
		ID:         row.ID,
		Repo:       row.Repo,
		Tag:        row.Tag,
		Title:      row.Title,
		Notes:      row.Notes,
		Draft:      row.Draft,
		Prerelease: row.Prerelease,
		Target:     row.Target,
		Commit:     row.CommitSha,
		Author:     Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		Assets:     []Asset{},
		CreatedAt:  row.CreatedAt,
	}
	if row.PublishedAt.Valid {
		release.PublishedAt = &row.PublishedAt.Time
	}
	rows, err := q.ListAssets(ctx, row.ID)
	if err != nil {
		return release, err
	}
	for _, asset := range rows {
		release.Assets = append(release.Assets, assetFromRow(asset))
	}
	return release, nil
}

func newReleaseEvent(release Release) releaseEvent {
	names := []string{}
	for _, asset := range release.Assets {
		names = append(names, asset.Name)
	}
	return releaseEvent{
		Assets:     names,
		Commit:     release.Commit,
		Draft:      release.Draft,
		ID:         release.ID,
		Prerelease: release.Prerelease,
		Tag:        release.Tag,
		Title:      release.Title,
	}
}

func publishedAt(release Release) sql.NullTime {
	if release.PublishedAt == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *release.PublishedAt, Valid: true}
}

func (s *Store) CreateRelease(ctx context.Context, actor Actor, release Release) (Release, int64, error) {
	var created Release
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := q.CreateRelease(ctx, queries.CreateReleaseParams{
			Repo:        release.Repo,
			Tag:         release.Tag,
			Title:       release.Title,
			Notes:       release.Notes,
			Draft:       release.Draft,
			Prerelease:  release.Prerelease,
			Target:      release.Target,
			CommitSha:   release.Commit,
			User:        actor.User,
			Via:         actor.Via,
			Agent:       actor.Agent,
			Session:     actor.Session,
			CreatedAt:   now(),
			PublishedAt: publishedAt(release),
		})
		if IsUniqueViolation(err) {
			return ErrExists
		}
		if err != nil {
			return err
		}
		if created, err = loadRelease(ctx, q, row); err != nil {
			return err
		}
		action := "release.published"
		if created.Draft {
			action = "release.created"
		}
		eventID, err = addEvent(ctx, q, action, created.Repo, 0, actor, newReleaseEvent(created))
		return err
	})
	return created, eventID, err
}

func (s *Store) Release(ctx context.Context, repo string, id int64) (Release, error) {
	row, err := s.queries.GetRelease(ctx, queries.GetReleaseParams{Repo: repo, ID: id})
	if err != nil {
		return Release{}, notFound(err)
	}
	return loadRelease(ctx, s.queries, row)
}

func (s *Store) ReleaseByTag(ctx context.Context, repo, tag string) (Release, error) {
	row, err := s.queries.GetReleaseByTag(ctx, queries.GetReleaseByTagParams{Repo: repo, Tag: tag})
	if err != nil {
		return Release{}, notFound(err)
	}
	return loadRelease(ctx, s.queries, row)
}

func (s *Store) LatestRelease(ctx context.Context, repo string) (Release, error) {
	row, err := s.queries.LatestRelease(ctx, repo)
	if err != nil {
		return Release{}, notFound(err)
	}
	return loadRelease(ctx, s.queries, row)
}

func (s *Store) Releases(ctx context.Context, repo string) ([]Release, error) {
	rows, err := s.queries.ListReleases(ctx, repo)
	if err != nil {
		return nil, err
	}
	releases := []Release{}
	for _, row := range rows {
		release, err := loadRelease(ctx, s.queries, row)
		if err != nil {
			return nil, err
		}
		releases = append(releases, release)
	}
	return releases, nil
}

func (s *Store) UpdateRelease(
	ctx context.Context,
	actor Actor,
	release Release,
	action string,
) (Release, int64, error) {
	var updated Release
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := q.UpdateRelease(ctx, queries.UpdateReleaseParams{
			Title:       release.Title,
			Notes:       release.Notes,
			Draft:       release.Draft,
			Prerelease:  release.Prerelease,
			Target:      release.Target,
			CommitSha:   release.Commit,
			PublishedAt: publishedAt(release),
			ID:          release.ID,
		})
		if err != nil {
			return notFound(err)
		}
		if updated, err = loadRelease(ctx, q, row); err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, action, updated.Repo, 0, actor, newReleaseEvent(updated))
		return err
	})
	return updated, eventID, err
}

func (s *Store) DeleteRelease(ctx context.Context, actor Actor, release Release) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		if err := deleted(q.DeleteRelease(ctx, release.ID)); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(ctx, q, "release.deleted", release.Repo, 0, actor, newReleaseEvent(release))
		return err
	})
	return eventID, err
}

func newAssetEvent(release Release, asset Asset, change string) assetEvent {
	return assetEvent{
		Change:    change,
		Name:      asset.Name,
		ReleaseID: release.ID,
		SHA256:    asset.SHA256,
		Size:      asset.Size,
		Tag:       release.Tag,
	}
}

func (s *Store) AddAsset(
	ctx context.Context,
	actor Actor,
	release Release,
	asset Asset,
	clobber bool,
) (Asset, *Asset, int64, error) {
	var created Asset
	var replaced *Asset
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
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
			ReleaseID:   release.ID,
			Name:        asset.Name,
			Object:      asset.Object,
			Size:        asset.Size,
			Sha256:      asset.SHA256,
			ContentType: asset.ContentType,
			User:        actor.User,
			Via:         actor.Via,
			Agent:       actor.Agent,
			Session:     actor.Session,
			CreatedAt:   now(),
		})
		if err != nil {
			return notFound(err)
		}
		created = assetFromRow(row)
		data := newAssetEvent(release, created, "uploaded")
		eventID, err = addEvent(ctx, q, "release.asset", release.Repo, 0, actor, data)
		return err
	})
	return created, replaced, eventID, err
}

func (s *Store) DeleteAsset(ctx context.Context, actor Actor, release Release, name string) (Asset, int64, error) {
	var asset Asset
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := q.GetAsset(ctx, queries.GetAssetParams{ReleaseID: release.ID, Name: name})
		if err != nil {
			return notFound(err)
		}
		asset = assetFromRow(row)
		if err := deleted(q.DeleteAsset(ctx, row.ID)); err != nil {
			return err
		}
		data := newAssetEvent(release, asset, "deleted")
		eventID, err = addEvent(ctx, q, "release.asset", release.Repo, 0, actor, data)
		return err
	})
	return asset, eventID, err
}
