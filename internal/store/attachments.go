package store

import (
	"context"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

type Attachment struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	ContentType string    `json:"content_type"`
	Uploader    Actor     `json:"uploader"`
	CreatedAt   time.Time `json:"created_at"`
	URL         string    `json:"url"`
	Repo        string    `json:"-"`
	Object      string    `json:"-"`
}

func attachmentFromRow(row queries.Attachment) Attachment {
	return Attachment{
		ID:          row.ID,
		Name:        row.Name,
		Size:        row.Size,
		SHA256:      row.Sha256,
		ContentType: row.ContentType,
		Uploader:    Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		CreatedAt:   row.CreatedAt,
		Repo:        row.Repo,
		Object:      row.Object,
	}
}

func (s *Store) AddAttachment(ctx context.Context, actor Actor, attachment Attachment) (Attachment, error) {
	row, err := s.queries.CreateAttachment(ctx, queries.CreateAttachmentParams{
		Repo:        attachment.Repo,
		Name:        attachment.Name,
		Object:      attachment.Object,
		Size:        attachment.Size,
		Sha256:      attachment.SHA256,
		ContentType: attachment.ContentType,
		User:        actor.User,
		Via:         actor.Via,
		Agent:       actor.Agent,
		Session:     actor.Session,
		CreatedAt:   now(),
	})
	if err != nil {
		return Attachment{}, err
	}
	return attachmentFromRow(row), nil
}

func (s *Store) Attachment(ctx context.Context, repo string, id int64) (Attachment, error) {
	row, err := s.queries.GetAttachment(ctx, queries.GetAttachmentParams{Repo: repo, ID: id})
	if err != nil {
		return Attachment{}, notFound(err)
	}
	return attachmentFromRow(row), nil
}
