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

func loadPull(q *queries.Queries, item queries.Item, pull queries.Pull) (Pull, error) {
	loaded, err := loadItem(q, item)
	result := Pull{Item: loaded, Base: pull.Base, Head: pull.Head, HeadSHA: pull.HeadSha, BaseSHA: pull.BaseSha, MergeSHA: pull.MergeSha}
	if pull.MergedAt.Valid {
		result.MergedAt = &pull.MergedAt.Time
	}
	return result, err
}

func getPull(q *queries.Queries, repo string, number int64) (queries.GetPullItemRow, error) {
	row, err := q.GetPullItem(context.Background(), queries.GetPullItemParams{Repo: repo, Number: number})
	if err != nil {
		return row, notFound(err)
	}
	if row.Item.State != "open" {
		return row, InvalidError(fmt.Sprintf("pull request #%d is %s", number, row.Item.State))
	}
	return row, nil
}

func (s *Store) CreatePull(repo string, actor Actor, title, body, base, head, headSHA, baseSHA string, labels, assignees []string) (Pull, int64, error) {
	var pull Pull
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		open, err := q.ListPullItems(ctx, queries.ListPullItemsParams{Repo: repo, State: "open"})
		if err != nil {
			return err
		}
		for _, existing := range open {
			if existing.Pull.Base == base && existing.Pull.Head == head {
				return InvalidError(fmt.Sprintf("pull request #%d already proposes %s into %s", existing.Item.Number, head, base))
			}
		}
		row, err := createItem(q, "pr", repo, actor, title, body, labels, assignees)
		if err != nil {
			return err
		}
		created := queries.CreatePullParams{ItemID: row.ID, Base: base, Head: head, HeadSha: headSHA, BaseSha: baseSHA}
		if err := q.CreatePull(ctx, created); err != nil {
			return err
		}
		if pull, err = loadPull(q, row, queries.Pull{ItemID: row.ID, Base: base, Head: head, HeadSha: headSHA, BaseSha: baseSHA}); err != nil {
			return err
		}
		eventID, err = addEvent(q, "pr.opened", repo, row.Number, actor, map[string]any{
			"number": row.Number, "title": title, "base": base, "head": head, "head_sha": headSHA, "labels": pull.Labels, "assignees": pull.Assignees,
		})
		return err
	})
	return pull, eventID, err
}

func (s *Store) Pull(repo string, number int64) (Pull, error) {
	row, err := s.queries.GetPullItem(context.Background(), queries.GetPullItemParams{Repo: repo, Number: number})
	if err != nil {
		return Pull{}, notFound(err)
	}
	return loadPull(s.queries, row.Item, row.Pull)
}

func (s *Store) Pulls(repo, state string) ([]Pull, error) {
	rows, err := s.queries.ListPullItems(context.Background(), queries.ListPullItemsParams{Repo: repo, State: state})
	if err != nil {
		return nil, err
	}
	pulls := []Pull{}
	for _, row := range rows {
		pull, err := loadPull(s.queries, row.Item, row.Pull)
		if err != nil {
			return nil, err
		}
		pulls = append(pulls, pull)
	}
	return pulls, nil
}

func (s *Store) SyncPull(repo string, number int64, actor Actor, headSHA, baseSHA string) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		row, err := getPull(q, repo, number)
		if err != nil {
			return err
		}
		if err := q.UpdatePullHead(ctx, queries.UpdatePullHeadParams{HeadSha: headSHA, BaseSha: baseSHA, ItemID: row.Item.ID}); err != nil {
			return err
		}
		if err := q.TouchItem(ctx, queries.TouchItemParams{UpdatedAt: now(), ID: row.Item.ID}); err != nil {
			return err
		}
		eventID, err = addEvent(q, "pr.synchronized", repo, number, actor, map[string]any{"number": number, "before": row.Pull.HeadSha, "after": headSHA})
		return err
	})
	return eventID, err
}

func (s *Store) MarkPullMerged(repo string, number int64, actor Actor, mergeSHA, baseSHA, method string) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		row, err := getPull(q, repo, number)
		if err != nil {
			return err
		}
		mergedAt := sql.NullTime{Time: now(), Valid: true}
		if err := q.MergePull(ctx, queries.MergePullParams{MergeSha: mergeSHA, BaseSha: baseSHA, MergedAt: mergedAt, ItemID: row.Item.ID}); err != nil {
			return err
		}
		update := queries.UpdateItemParams{ID: row.Item.ID, Title: row.Item.Title, Body: row.Item.Body, State: "merged", UpdatedAt: mergedAt.Time, ClosedAt: mergedAt}
		if _, err := q.UpdateItem(ctx, update); err != nil {
			return err
		}
		eventID, err = addEvent(q, "pr.merged", repo, number, actor, map[string]any{"number": number, "sha": mergeSHA, "base": row.Pull.Base, "method": method})
		return err
	})
	return eventID, err
}
