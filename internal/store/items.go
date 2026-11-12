package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/muchobien/lyra/internal/store/queries"
)

type InvalidError string

func (invalid InvalidError) Error() string {
	return string(invalid)
}

type Item struct {
	Number       int64      `json:"number"`
	Repo         string     `json:"repo"`
	Kind         string     `json:"kind"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	State        string     `json:"state"`
	Author       Actor      `json:"author"`
	Labels       []string   `json:"labels"`
	Assignees    []string   `json:"assignees"`
	CommentCount int64      `json:"comment_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at"`
	id           int64
}

type Comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Author    Actor     `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

type Label = queries.Label

type ItemEdit struct {
	Title           *string
	Body            *string
	State           *string
	AddLabels       []string
	RemoveLabels    []string
	AddAssignees    []string
	RemoveAssignees []string
	Commit          string
}

var colorPattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

func checkName(kind, name string) error {
	if name == "" || len(name) > 50 || strings.TrimSpace(name) != name || strings.ContainsAny(name, ",/") || strings.ContainsFunc(name, unicode.IsControl) {
		return InvalidError(fmt.Sprintf("invalid %s %q: 1 to 50 characters, no commas, slashes or surrounding spaces", kind, name))
	}
	return nil
}

func loadItem(q *queries.Queries, row queries.Item) (Item, error) {
	ctx := context.Background()
	item := Item{
		Number: row.Number, Repo: row.Repo, Kind: row.Kind, Title: row.Title, Body: row.Body, State: row.State,
		Author:    Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Labels: []string{}, Assignees: []string{}, id: row.ID,
	}
	if row.ClosedAt.Valid {
		item.ClosedAt = &row.ClosedAt.Time
	}
	labels, err := q.ListItemLabels(ctx, row.ID)
	if err != nil {
		return item, err
	}
	assignees, err := q.ListItemAssignees(ctx, row.ID)
	if err != nil {
		return item, err
	}
	item.Labels = append(item.Labels, labels...)
	item.Assignees = append(item.Assignees, assignees...)
	item.CommentCount, err = q.CountComments(ctx, row.ID)
	return item, err
}

func getItem(q *queries.Queries, kind, repo string, number int64) (queries.Item, error) {
	row, err := q.GetItem(context.Background(), queries.GetItemParams{Repo: repo, Number: number})
	if err != nil {
		return row, notFound(err)
	}
	if row.Kind != kind {
		return row, ErrNotFound
	}
	return row, nil
}

func applyLabel(q *queries.Queries, item queries.Item, name string, add bool) (bool, error) {
	if err := checkName("label", name); err != nil {
		return false, err
	}
	ctx := context.Background()
	label, err := q.GetLabel(ctx, queries.GetLabelParams{Repo: item.Repo, Name: name})
	if errors.Is(err, sql.ErrNoRows) {
		return false, InvalidError(fmt.Sprintf("label %q does not exist in %s; create it with lyra label create", name, item.Repo))
	}
	if err != nil {
		return false, err
	}
	var count int64
	if add {
		count, err = q.AddItemLabel(ctx, queries.AddItemLabelParams{ItemID: item.ID, LabelID: label.ID})
	} else {
		count, err = q.RemoveItemLabel(ctx, queries.RemoveItemLabelParams{ItemID: item.ID, LabelID: label.ID})
	}
	return count > 0, err
}

func applyAssignee(q *queries.Queries, item queries.Item, assignee string, add bool) (bool, error) {
	if err := checkName("assignee", assignee); err != nil {
		return false, err
	}
	ctx := context.Background()
	var count int64
	var err error
	if add {
		count, err = q.AddItemAssignee(ctx, queries.AddItemAssigneeParams{ItemID: item.ID, Assignee: assignee})
	} else {
		count, err = q.RemoveItemAssignee(ctx, queries.RemoveItemAssigneeParams{ItemID: item.ID, Assignee: assignee})
	}
	return count > 0, err
}

func commentFromRow(row queries.Comment) Comment {
	return Comment{ID: row.ID, Body: row.Body, Author: Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session}, CreatedAt: row.CreatedAt}
}

func createItem(q *queries.Queries, kind, repo string, actor Actor, title, body string, labels, assignees []string) (queries.Item, error) {
	row, err := q.CreateItem(context.Background(), queries.CreateItemParams{
		Repo: repo, Kind: kind, Title: title, Body: body, User: actor.User, Via: actor.Via, Agent: actor.Agent, Session: actor.Session, CreatedAt: now(),
	})
	if err != nil {
		return row, err
	}
	for _, name := range labels {
		if _, err := applyLabel(q, row, name, true); err != nil {
			return row, err
		}
	}
	for _, assignee := range assignees {
		if _, err := applyAssignee(q, row, assignee, true); err != nil {
			return row, err
		}
	}
	return row, nil
}

func (s *Store) CreateItem(kind, repo string, actor Actor, title, body string, labels, assignees []string) (Item, int64, error) {
	var item Item
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		row, err := createItem(q, kind, repo, actor, title, body, labels, assignees)
		if err != nil {
			return err
		}
		if item, err = loadItem(q, row); err != nil {
			return err
		}
		eventID, err = addEvent(q, kind+".opened", repo, row.Number, actor, map[string]any{"number": row.Number, "title": title, "labels": item.Labels, "assignees": item.Assignees})
		return err
	})
	return item, eventID, err
}

func (s *Store) Item(kind, repo string, number int64) (Item, error) {
	row, err := getItem(s.queries, kind, repo, number)
	if err != nil {
		return Item{}, err
	}
	return loadItem(s.queries, row)
}

func (s *Store) Items(kind, repo, state string) ([]Item, error) {
	rows, err := s.queries.ListItems(context.Background(), queries.ListItemsParams{Repo: repo, Kind: kind, State: state})
	if err != nil {
		return nil, err
	}
	items := []Item{}
	for _, row := range rows {
		item, err := loadItem(s.queries, row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) EditItem(kind, repo string, number int64, actor Actor, edit ItemEdit) (Item, int64, error) {
	var item Item
	var lastEvent int64
	err := s.inTx(func(q *queries.Queries) error {
		row, err := getItem(q, kind, repo, number)
		if err != nil {
			return err
		}
		record := func(action string, data map[string]any) error {
			data["number"] = number
			lastEvent, err = addEvent(q, kind+"."+action, repo, number, actor, data)
			return err
		}
		update := queries.UpdateItemParams{ID: row.ID, Title: row.Title, Body: row.Body, State: row.State, ClosedAt: row.ClosedAt}
		changes := map[string]any{}
		if edit.Title != nil && *edit.Title != row.Title {
			changes["title"] = map[string]string{"from": row.Title}
			update.Title = *edit.Title
		}
		if edit.Body != nil && *edit.Body != row.Body {
			changes["body"] = map[string]string{"from": row.Body}
			update.Body = *edit.Body
		}
		if len(changes) > 0 {
			if err := record("edited", map[string]any{"changes": changes}); err != nil {
				return err
			}
		}
		if edit.State != nil && *edit.State != row.State {
			switch {
			case *edit.State == "closed" && row.State == "open":
				update.State = "closed"
				update.ClosedAt = sql.NullTime{Time: now(), Valid: true}
				closedBy := map[string]any{}
				if edit.Commit != "" {
					closedBy["commit"] = edit.Commit
				}
				err = record("closed", closedBy)
			case *edit.State == "open" && row.State == "closed":
				update.State = "open"
				update.ClosedAt = sql.NullTime{}
				err = record("reopened", map[string]any{})
			case *edit.State == "open" || *edit.State == "closed":
				err = InvalidError(fmt.Sprintf("#%d is %s", number, row.State))
			default:
				err = InvalidError("state must be open or closed")
			}
			if err != nil {
				return err
			}
		}
		for _, change := range []struct {
			names  []string
			add    bool
			label  bool
			action string
		}{
			{edit.AddLabels, true, true, "labeled"},
			{edit.RemoveLabels, false, true, "unlabeled"},
			{edit.AddAssignees, true, false, "assigned"},
			{edit.RemoveAssignees, false, false, "unassigned"},
		} {
			for _, name := range change.names {
				var applied bool
				var key string
				if change.label {
					applied, err = applyLabel(q, row, name, change.add)
					key = "label"
				} else {
					applied, err = applyAssignee(q, row, name, change.add)
					key = "assignee"
				}
				if err != nil {
					return err
				}
				if applied {
					if err := record(change.action, map[string]any{key: name}); err != nil {
						return err
					}
				}
			}
		}
		if lastEvent != 0 {
			update.UpdatedAt = now()
			if row, err = q.UpdateItem(context.Background(), update); err != nil {
				return err
			}
		}
		item, err = loadItem(q, row)
		return err
	})
	return item, lastEvent, err
}

func (s *Store) AddComment(kind, repo string, number int64, actor Actor, body string) (Comment, int64, error) {
	var comment Comment
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
		row, err := getItem(q, kind, repo, number)
		if err != nil {
			return err
		}
		created, err := q.CreateComment(ctx, queries.CreateCommentParams{
			ItemID: row.ID, Body: body, User: actor.User, Via: actor.Via, Agent: actor.Agent, Session: actor.Session, CreatedAt: now(),
		})
		if err != nil {
			return err
		}
		if err := q.TouchItem(ctx, queries.TouchItemParams{UpdatedAt: created.CreatedAt, ID: row.ID}); err != nil {
			return err
		}
		comment = commentFromRow(created)
		eventID, err = addEvent(q, kind+".comment", repo, number, actor, map[string]any{"number": number, "comment": created.ID, "body": body})
		return err
	})
	return comment, eventID, err
}

func (s *Store) Comments(item Item) ([]Comment, error) {
	rows, err := s.queries.ListComments(context.Background(), item.id)
	if err != nil {
		return nil, err
	}
	comments := []Comment{}
	for _, row := range rows {
		comments = append(comments, commentFromRow(row))
	}
	return comments, nil
}

func (s *Store) CreateLabel(repo string, actor Actor, name, color, description string) (Label, int64, error) {
	if err := checkName("label", name); err != nil {
		return Label{}, 0, err
	}
	color = strings.ToLower(strings.TrimPrefix(color, "#"))
	if color != "" && !colorPattern.MatchString(color) {
		return Label{}, 0, InvalidError("color must be six hex digits, such as d73a4a")
	}
	var label Label
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		var err error
		label, err = q.CreateLabel(context.Background(), queries.CreateLabelParams{Repo: repo, Name: name, Color: color, Description: description})
		if err != nil {
			return err
		}
		eventID, err = addEvent(q, "label.create", repo, 0, actor, map[string]any{"name": name, "color": color, "description": description})
		return err
	})
	return label, eventID, err
}

func (s *Store) Labels(repo string) ([]Label, error) {
	labels, err := s.queries.ListLabels(context.Background(), repo)
	if labels == nil {
		labels = []Label{}
	}
	return labels, err
}

func (s *Store) DeleteLabel(repo string, actor Actor, name string) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		if err := deleted(q.DeleteLabel(context.Background(), queries.DeleteLabelParams{Repo: repo, Name: name})); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(q, "label.delete", repo, 0, actor, map[string]any{"name": name})
		return err
	})
	return eventID, err
}
