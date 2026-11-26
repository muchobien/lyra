package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

type Artifact struct {
	ID        int64      `json:"id"`
	JobID     int64      `json:"job_id"`
	Name      string     `json:"name"`
	Size      int64      `json:"size"`
	SHA256    string     `json:"sha256"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	RunID     int64      `json:"-"`
	Repo      string     `json:"-"`
	Object    string     `json:"-"`
	Status    string     `json:"-"`
}

func artifactFromRow(row queries.Artifact) Artifact {
	return Artifact{
		ID:        row.ID,
		JobID:     row.JobID,
		Name:      row.Name,
		Size:      row.Size,
		SHA256:    row.Sha256,
		CreatedAt: row.CreatedAt,
		ExpiresAt: nullTime(row.ExpiresAt),
		RunID:     row.RunID,
		Object:    row.Object,
		Status:    row.Status,
	}
}

func (s *Store) CreateArtifact(ctx context.Context, artifact Artifact) (Artifact, *Artifact, error) {
	var created Artifact
	var replaced *Artifact
	err := s.inTx(ctx, func(q *queries.Queries) error {
		existing, err := q.GetArtifactByName(ctx, queries.GetArtifactByNameParams{
			RunID: artifact.RunID,
			Name:  artifact.Name,
		})
		switch {
		case err == nil && existing.Status == "ready" && existing.JobID != artifact.JobID:
			return ErrExists
		case err == nil:
			old := artifactFromRow(existing)
			replaced = &old
			if err := deleted(q.DeleteArtifact(ctx, existing.ID)); err != nil {
				return err
			}
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}
		expires := sql.NullTime{}
		if artifact.ExpiresAt != nil {
			expires = sql.NullTime{Time: artifact.ExpiresAt.UTC(), Valid: true}
		}
		row, err := q.CreateArtifact(ctx, queries.CreateArtifactParams{
			RunID:     artifact.RunID,
			JobID:     artifact.JobID,
			Name:      artifact.Name,
			Object:    artifact.Object,
			CreatedAt: now(),
			ExpiresAt: expires,
		})
		created = artifactFromRow(row)
		return err
	})
	return created, replaced, err
}

func (s *Store) Artifact(ctx context.Context, runID int64, name string) (Artifact, error) {
	row, err := s.queries.GetArtifactByName(ctx, queries.GetArtifactByNameParams{RunID: runID, Name: name})
	if err != nil {
		return Artifact{}, notFound(err)
	}
	return artifactFromRow(row), nil
}

func (s *Store) ArtifactByID(ctx context.Context, id int64) (Artifact, error) {
	row, err := s.queries.GetArtifact(ctx, id)
	if err != nil {
		return Artifact{}, notFound(err)
	}
	return artifactFromRow(row), nil
}

func (s *Store) Artifacts(ctx context.Context, runID int64) ([]Artifact, error) {
	rows, err := s.queries.ListRunArtifacts(ctx, runID)
	if err != nil {
		return nil, err
	}
	artifacts := []Artifact{}
	for _, row := range rows {
		artifacts = append(artifacts, artifactFromRow(row))
	}
	return artifacts, nil
}

func (s *Store) UpdateArtifact(ctx context.Context, artifact Artifact) error {
	return s.queries.UpdateArtifact(ctx, queries.UpdateArtifactParams{
		Size:   artifact.Size,
		Sha256: artifact.SHA256,
		Status: artifact.Status,
		ID:     artifact.ID,
	})
}

func (s *Store) DeleteArtifact(ctx context.Context, id int64) error {
	return deleted(s.queries.DeleteArtifact(ctx, id))
}

func (s *Store) ExpiredArtifacts(ctx context.Context, before, pendingBefore time.Time) ([]Artifact, error) {
	rows, err := s.queries.ListExpiredArtifacts(ctx, queries.ListExpiredArtifactsParams{
		ExpiresAt: sql.NullTime{Time: before, Valid: true},
		CreatedAt: pendingBefore,
	})
	if err != nil {
		return nil, err
	}
	artifacts := []Artifact{}
	for _, row := range rows {
		artifact := artifactFromRow(queries.Artifact{
			ID:        row.ID,
			RunID:     row.RunID,
			JobID:     row.JobID,
			Name:      row.Name,
			Object:    row.Object,
			Size:      row.Size,
			Sha256:    row.Sha256,
			Status:    row.Status,
			CreatedAt: row.CreatedAt,
			ExpiresAt: row.ExpiresAt,
		})
		artifact.Repo = row.Repo
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}
