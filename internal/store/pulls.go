package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

type Pull struct {
	Item
	Base     string     `json:"base"`
	Head     string     `json:"head"`
	HeadSHA  string     `json:"head_sha"`
	BaseSHA  string     `json:"base_sha"`
	MergeSHA string     `json:"merge_sha,omitempty"`
	MergedAt *time.Time `json:"merged_at"`
}

type pullOpenedEvent struct {
	Assignees []string `json:"assignees"`
	Base      string   `json:"base"`
	Head      string   `json:"head"`
	HeadSHA   string   `json:"head_sha"`
	Labels    []string `json:"labels"`
	Number    int64    `json:"number"`
	Title     string   `json:"title"`
}

type pullSynchronizedEvent struct {
	After  string `json:"after"`
	Before string `json:"before"`
	Number int64  `json:"number"`
}

type pullMergedEvent struct {
	Base   string `json:"base"`
	Method string `json:"method"`
	Number int64  `json:"number"`
	SHA    string `json:"sha"`
}

func loadPull(ctx context.Context, q *queries.Queries, item queries.Item, pull queries.Pull) (Pull, error) {
	loaded, err := loadItem(ctx, q, item)
	result := Pull{
		Item:     loaded,
		Base:     pull.Base,
		Head:     pull.Head,
		HeadSHA:  pull.HeadSha,
		BaseSHA:  pull.BaseSha,
		MergeSHA: pull.MergeSha,
	}
	if pull.MergedAt.Valid {
		result.MergedAt = &pull.MergedAt.Time
	}
	return result, err
}

func getPull(ctx context.Context, q *queries.Queries, repo string, number int64) (queries.GetPullItemRow, error) {
	row, err := q.GetPullItem(ctx, queries.GetPullItemParams{Repo: repo, Number: number})
	if err != nil {
		return row, notFound(err)
	}
	if row.Item.State != "open" {
		return row, InvalidError(fmt.Sprintf("pull request #%d is %s", number, row.Item.State))
	}
	return row, nil
}

func (s *Store) CreatePull(
	ctx context.Context,
	repo string,
	actor Actor,
	title, body, base, head, headSHA, baseSHA string,
	labels, assignees []string,
) (Pull, int64, error) {
	var pull Pull
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		open, err := q.ListPullItems(ctx, queries.ListPullItemsParams{Repo: repo, State: "open"})
		if err != nil {
			return err
		}
		for _, existing := range open {
			if existing.Pull.Base != base || existing.Pull.Head != head {
				continue
			}
			number := existing.Item.Number
			return InvalidError(fmt.Sprintf("pull request #%d already proposes %s into %s", number, head, base))
		}
		row, err := createItem(ctx, q, "pr", repo, actor, title, body, labels, assignees)
		if err != nil {
			return err
		}
		created := queries.CreatePullParams{ItemID: row.ID, Base: base, Head: head, HeadSha: headSHA, BaseSha: baseSHA}
		if err := q.CreatePull(ctx, created); err != nil {
			return err
		}
		pull, err = loadPull(ctx, q, row, queries.Pull{
			ItemID:  row.ID,
			Base:    base,
			Head:    head,
			HeadSha: headSHA,
			BaseSha: baseSHA,
		})
		if err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, "pr.opened", repo, row.Number, actor, pullOpenedEvent{
			Assignees: pull.Assignees,
			Base:      base,
			Head:      head,
			HeadSHA:   headSHA,
			Labels:    pull.Labels,
			Number:    row.Number,
			Title:     title,
		})
		return err
	})
	return pull, eventID, err
}

func (s *Store) Pull(ctx context.Context, repo string, number int64) (Pull, error) {
	row, err := s.queries.GetPullItem(ctx, queries.GetPullItemParams{Repo: repo, Number: number})
	if err != nil {
		return Pull{}, notFound(err)
	}
	return loadPull(ctx, s.queries, row.Item, row.Pull)
}

func (s *Store) Pulls(ctx context.Context, repo, state string) ([]Pull, error) {
	rows, err := s.queries.ListPullItems(ctx, queries.ListPullItemsParams{Repo: repo, State: state})
	if err != nil {
		return nil, err
	}
	pulls := []Pull{}
	for _, row := range rows {
		pull, err := loadPull(ctx, s.queries, row.Item, row.Pull)
		if err != nil {
			return nil, err
		}
		pulls = append(pulls, pull)
	}
	return pulls, nil
}

func (s *Store) SyncPull(
	ctx context.Context,
	repo string,
	number int64,
	actor Actor,
	headSHA, baseSHA string,
) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := getPull(ctx, q, repo, number)
		if err != nil {
			return err
		}
		head := queries.UpdatePullHeadParams{HeadSha: headSHA, BaseSha: baseSHA, ItemID: row.Item.ID}
		if err := q.UpdatePullHead(ctx, head); err != nil {
			return err
		}
		if err := q.TouchItem(ctx, queries.TouchItemParams{UpdatedAt: now(), ID: row.Item.ID}); err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, "pr.synchronized", repo, number, actor, pullSynchronizedEvent{
			After:  headSHA,
			Before: row.Pull.HeadSha,
			Number: number,
		})
		return err
	})
	return eventID, err
}

func (s *Store) MarkPullMerged(
	ctx context.Context,
	repo string,
	number int64,
	actor Actor,
	mergeSHA, baseSHA, method string,
) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := getPull(ctx, q, repo, number)
		if err != nil {
			return err
		}
		mergedAt := sql.NullTime{Time: now(), Valid: true}
		err = q.MergePull(ctx, queries.MergePullParams{
			MergeSha: mergeSHA,
			BaseSha:  baseSHA,
			MergedAt: mergedAt,
			ItemID:   row.Item.ID,
		})
		if err != nil {
			return err
		}
		update := queries.UpdateItemParams{
			ID:        row.Item.ID,
			Title:     row.Item.Title,
			Body:      row.Item.Body,
			State:     "merged",
			UpdatedAt: mergedAt.Time,
			ClosedAt:  mergedAt,
		}
		if _, err := q.UpdateItem(ctx, update); err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, "pr.merged", repo, number, actor, pullMergedEvent{
			Base:   row.Pull.Base,
			Method: method,
			Number: number,
			SHA:    mergeSHA,
		})
		return err
	})
	return eventID, err
}
