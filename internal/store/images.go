package store

import (
	"context"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

type Image struct {
	Repo      string    `json:"repo"`
	Name      string    `json:"name"`
	Tag       string    `json:"tag"`
	Digest    string    `json:"digest"`
	MediaType string    `json:"media_type"`
	Pusher    Actor     `json:"pusher"`
	PushedAt  time.Time `json:"pushed_at"`
	Reference string    `json:"reference,omitempty"`
}

func imageFromRow(row queries.Image) Image {
	return Image{
		Repo: row.Repo, Name: row.Name, Tag: row.Tag, Digest: row.Digest, MediaType: row.MediaType,
		Pusher: Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session}, PushedAt: row.PushedAt,
	}
}

func (s *Store) SetImageTag(actor Actor, image Image) (Image, int64, error) {
	var saved Image
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		row, err := q.SetImageTag(context.Background(), queries.SetImageTagParams{
			Repo: image.Repo, Name: image.Name, Tag: image.Tag, Digest: image.Digest, MediaType: image.MediaType,
			User: actor.User, Via: actor.Via, Agent: actor.Agent, Session: actor.Session, PushedAt: now(),
		})
		if err != nil {
			return err
		}
		saved = imageFromRow(row)
		eventID, err = addEvent(q, "image.pushed", image.Repo, 0, actor, map[string]any{
			"name": saved.Name, "tag": saved.Tag, "digest": saved.Digest, "media_type": saved.MediaType,
		})
		return err
	})
	return saved, eventID, err
}

func (s *Store) DeleteImage(actor Actor, repo, name, reference string, byDigest bool) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		var tags []string
		var err error
		if byDigest {
			tags, err = q.DeleteImageDigest(ctx, queries.DeleteImageDigestParams{Name: name, Digest: reference})
		} else {
			tags, err = q.DeleteImageTag(ctx, queries.DeleteImageTagParams{Name: name, Tag: reference})
		}
		if err != nil {
			return err
		}
		if tags == nil {
			tags = []string{}
		}
		eventID, err = addEvent(q, "image.deleted", repo, 0, actor, map[string]any{"name": name, "reference": reference, "tags": tags})
		return err
	})
	return eventID, err
}

func (s *Store) Images(repo string) ([]Image, error) {
	rows, err := s.queries.ListImages(context.Background(), repo)
	if err != nil {
		return nil, err
	}
	images := []Image{}
	for _, row := range rows {
		images = append(images, imageFromRow(row))
	}
	return images, nil
}
