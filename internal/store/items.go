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

type LineRef struct {
	CommitSHA string `json:"commit_sha,omitempty"`
	Path      string `json:"path,omitempty"`
	Side      string `json:"side,omitempty"`
	Line      int64  `json:"line,omitzero"`
}

type Comment struct {
	ID       int64  `json:"id"`
	Body     string `json:"body"`
	Author   Actor  `json:"author"`
	ReviewID int64  `json:"review_id,omitzero"`
	LineRef
	Outdated  bool      `json:"outdated,omitzero"`
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

type itemOpenedEvent struct {
	Assignees []string `json:"assignees"`
	Labels    []string `json:"labels"`
	Number    int64    `json:"number"`
	Title     string   `json:"title"`
}

type previousValue struct {
	From string `json:"from"`
}

type itemChanges struct {
	Body  *previousValue `json:"body,omitzero"`
	Title *previousValue `json:"title,omitzero"`
}

type itemEditedEvent struct {
	Changes itemChanges `json:"changes"`
	Number  int64       `json:"number"`
}

type itemClosedEvent struct {
	Commit string `json:"commit,omitempty"`
	Number int64  `json:"number"`
}

type itemLabelEvent struct {
	Label  string `json:"label"`
	Number int64  `json:"number"`
}

type itemAssigneeEvent struct {
	Assignee string `json:"assignee"`
	Number   int64  `json:"number"`
}

type commentEvent struct {
	Body    string  `json:"body"`
	Comment int64   `json:"comment"`
	Line    *int64  `json:"line,omitzero"`
	Number  int64   `json:"number"`
	Path    string  `json:"path,omitempty"`
	Side    *string `json:"side,omitzero"`
}

type labelCreatedEvent struct {
	Color       string `json:"color"`
	Description string `json:"description"`
	Name        string `json:"name"`
}

type recordEvent func(action string, data any) error

var colorPattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

func checkName(kind, name string) error {
	valid := name != "" &&
		len(name) <= 50 &&
		strings.TrimSpace(name) == name &&
		!strings.ContainsAny(name, ",/") &&
		!strings.ContainsFunc(name, unicode.IsControl)
	if !valid {
		return InvalidError(fmt.Sprintf(
			"invalid %s %q: 1 to 50 characters, no commas, slashes or surrounding spaces",
			kind,
			name,
		))
	}
	return nil
}

func loadItem(ctx context.Context, q *queries.Queries, row queries.Item) (Item, error) {
	item := Item{
		Number:    row.Number,
		Repo:      row.Repo,
		Kind:      row.Kind,
		Title:     row.Title,
		Body:      row.Body,
		State:     row.State,
		Author:    Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Labels:    []string{},
		Assignees: []string{},
		id:        row.ID,
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

func getItem(ctx context.Context, q *queries.Queries, kind, repo string, number int64) (queries.Item, error) {
	row, err := q.GetItem(ctx, queries.GetItemParams{Repo: repo, Number: number})
	if err != nil {
		return row, notFound(err)
	}
	if row.Kind != kind {
		return row, ErrNotFound
	}
	return row, nil
}

func applyLabel(ctx context.Context, q *queries.Queries, item queries.Item, name string, add bool) (bool, error) {
	if err := checkName("label", name); err != nil {
		return false, err
	}
	label, err := q.GetLabel(ctx, queries.GetLabelParams{Repo: item.Repo, Name: name})
	if errors.Is(err, sql.ErrNoRows) {
		return false, InvalidError(fmt.Sprintf(
			"label %q does not exist in %s; create it with lyra label create",
			name,
			item.Repo,
		))
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

func applyAssignee(
	ctx context.Context,
	q *queries.Queries,
	item queries.Item,
	assignee string,
	add bool,
) (bool, error) {
	if err := checkName("assignee", assignee); err != nil {
		return false, err
	}
	var count int64
	var err error
	if add {
		count, err = q.AddItemAssignee(ctx, queries.AddItemAssigneeParams{ItemID: item.ID, Assignee: assignee})
	} else {
		count, err = q.RemoveItemAssignee(ctx, queries.RemoveItemAssigneeParams{
			ItemID:   item.ID,
			Assignee: assignee,
		})
	}
	return count > 0, err
}

func commentFromRow(row queries.Comment) Comment {
	return Comment{
		ID:        row.ID,
		Body:      row.Body,
		Author:    Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		ReviewID:  row.ReviewID.Int64,
		LineRef:   LineRef{CommitSHA: row.CommitSha, Path: row.Path, Side: row.Side, Line: row.Line},
		CreatedAt: row.CreatedAt,
	}
}

func createComment(
	ctx context.Context,
	q *queries.Queries,
	itemID int64,
	actor Actor,
	body string,
	reviewID int64,
	line LineRef,
	at time.Time,
) (queries.Comment, error) {
	return q.CreateComment(ctx, queries.CreateCommentParams{
		ItemID:    itemID,
		Body:      body,
		User:      actor.User,
		Via:       actor.Via,
		Agent:     actor.Agent,
		Session:   actor.Session,
		CreatedAt: at,
		ReviewID:  sql.NullInt64{Int64: reviewID, Valid: reviewID != 0},
		CommitSha: line.CommitSHA,
		Path:      line.Path,
		Side:      line.Side,
		Line:      line.Line,
	})
}

func createItem(
	ctx context.Context,
	q *queries.Queries,
	kind, repo string,
	actor Actor,
	title, body string,
	labels, assignees []string,
) (queries.Item, error) {
	row, err := q.CreateItem(ctx, queries.CreateItemParams{
		Repo:      repo,
		Kind:      kind,
		Title:     title,
		Body:      body,
		User:      actor.User,
		Via:       actor.Via,
		Agent:     actor.Agent,
		Session:   actor.Session,
		CreatedAt: now(),
	})
	if err != nil {
		return row, err
	}
	for _, name := range labels {
		if _, err := applyLabel(ctx, q, row, name, true); err != nil {
			return row, err
		}
	}
	for _, assignee := range assignees {
		if _, err := applyAssignee(ctx, q, row, assignee, true); err != nil {
			return row, err
		}
	}
	return row, nil
}

func (s *Store) CreateItem(
	ctx context.Context,
	kind, repo string,
	actor Actor,
	title, body string,
	labels, assignees []string,
) (Item, int64, error) {
	var item Item
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := createItem(ctx, q, kind, repo, actor, title, body, labels, assignees)
		if err != nil {
			return err
		}
		if item, err = loadItem(ctx, q, row); err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, kind+".opened", repo, row.Number, actor, itemOpenedEvent{
			Assignees: item.Assignees,
			Labels:    item.Labels,
			Number:    row.Number,
			Title:     title,
		})
		return err
	})
	return item, eventID, err
}

func (s *Store) Item(ctx context.Context, kind, repo string, number int64) (Item, error) {
	row, err := getItem(ctx, s.queries, kind, repo, number)
	if err != nil {
		return Item{}, err
	}
	return loadItem(ctx, s.queries, row)
}

func (s *Store) Items(ctx context.Context, kind, repo, state string) ([]Item, error) {
	rows, err := s.queries.ListItems(ctx, queries.ListItemsParams{Repo: repo, Kind: kind, State: state})
	if err != nil {
		return nil, err
	}
	items := []Item{}
	for _, row := range rows {
		item, err := loadItem(ctx, s.queries, row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func editText(row queries.Item, edit ItemEdit, update *queries.UpdateItemParams, record recordEvent) error {
	changes := itemChanges{}
	if edit.Title != nil && *edit.Title != row.Title {
		changes.Title = &previousValue{From: row.Title}
		update.Title = *edit.Title
	}
	if edit.Body != nil && *edit.Body != row.Body {
		changes.Body = &previousValue{From: row.Body}
		update.Body = *edit.Body
	}
	if changes.Title == nil && changes.Body == nil {
		return nil
	}
	return record("edited", itemEditedEvent{Changes: changes, Number: row.Number})
}

func editState(row queries.Item, edit ItemEdit, update *queries.UpdateItemParams, record recordEvent) error {
	if edit.State == nil || *edit.State == row.State {
		return nil
	}
	switch {
	case *edit.State == "closed" && row.State == "open":
		update.State = "closed"
		update.ClosedAt = sql.NullTime{Time: now(), Valid: true}
		return record("closed", itemClosedEvent{Commit: edit.Commit, Number: row.Number})
	case *edit.State == "open" && row.State == "closed":
		update.State = "open"
		update.ClosedAt = sql.NullTime{}
		return record("reopened", numberEvent{Number: row.Number})
	case *edit.State == "open" || *edit.State == "closed":
		return InvalidError(fmt.Sprintf("#%d is %s", row.Number, row.State))
	default:
		return InvalidError("state must be open or closed")
	}
}

func editLabels(
	ctx context.Context,
	q *queries.Queries,
	row queries.Item,
	names []string,
	add bool,
	record recordEvent,
) error {
	action := "unlabeled"
	if add {
		action = "labeled"
	}
	for _, name := range names {
		applied, err := applyLabel(ctx, q, row, name, add)
		if err != nil {
			return err
		}
		if !applied {
			continue
		}
		if err := record(action, itemLabelEvent{Label: name, Number: row.Number}); err != nil {
			return err
		}
	}
	return nil
}

func editAssignees(
	ctx context.Context,
	q *queries.Queries,
	row queries.Item,
	names []string,
	add bool,
	record recordEvent,
) error {
	action := "unassigned"
	if add {
		action = "assigned"
	}
	for _, name := range names {
		applied, err := applyAssignee(ctx, q, row, name, add)
		if err != nil {
			return err
		}
		if !applied {
			continue
		}
		if err := record(action, itemAssigneeEvent{Assignee: name, Number: row.Number}); err != nil {
			return err
		}
	}
	return nil
}

func applyEdit(
	ctx context.Context,
	q *queries.Queries,
	row queries.Item,
	edit ItemEdit,
	update *queries.UpdateItemParams,
	record recordEvent,
) error {
	if err := editText(row, edit, update, record); err != nil {
		return err
	}
	if err := editState(row, edit, update, record); err != nil {
		return err
	}
	if err := editLabels(ctx, q, row, edit.AddLabels, true, record); err != nil {
		return err
	}
	if err := editLabels(ctx, q, row, edit.RemoveLabels, false, record); err != nil {
		return err
	}
	if err := editAssignees(ctx, q, row, edit.AddAssignees, true, record); err != nil {
		return err
	}
	return editAssignees(ctx, q, row, edit.RemoveAssignees, false, record)
}

func (s *Store) EditItem(
	ctx context.Context,
	kind, repo string,
	number int64,
	actor Actor,
	edit ItemEdit,
) (Item, int64, error) {
	var item Item
	var lastEvent int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := getItem(ctx, q, kind, repo, number)
		if err != nil {
			return err
		}
		record := func(action string, data any) error {
			var err error
			lastEvent, err = addEvent(ctx, q, kind+"."+action, repo, number, actor, data)
			return err
		}
		update := queries.UpdateItemParams{
			ID:       row.ID,
			Title:    row.Title,
			Body:     row.Body,
			State:    row.State,
			ClosedAt: row.ClosedAt,
		}
		if err := applyEdit(ctx, q, row, edit, &update, record); err != nil {
			return err
		}
		if lastEvent != 0 {
			update.UpdatedAt = now()
			if row, err = q.UpdateItem(ctx, update); err != nil {
				return err
			}
		}
		item, err = loadItem(ctx, q, row)
		return err
	})
	return item, lastEvent, err
}

func (s *Store) AddComment(
	ctx context.Context,
	kind, repo string,
	number int64,
	actor Actor,
	body string,
	line LineRef,
) (Comment, int64, error) {
	var comment Comment
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := getItem(ctx, q, kind, repo, number)
		if err != nil {
			return err
		}
		created, err := createComment(ctx, q, row.ID, actor, body, 0, line, now())
		if err != nil {
			return err
		}
		if err := q.TouchItem(ctx, queries.TouchItemParams{UpdatedAt: created.CreatedAt, ID: row.ID}); err != nil {
			return err
		}
		comment = commentFromRow(created)
		data := commentEvent{Body: body, Comment: created.ID, Number: number}
		if line.Path != "" {
			data.Line, data.Path, data.Side = &line.Line, line.Path, &line.Side
		}
		eventID, err = addEvent(ctx, q, kind+".comment", repo, number, actor, data)
		return err
	})
	return comment, eventID, err
}

func (s *Store) Comments(ctx context.Context, item Item) ([]Comment, error) {
	rows, err := s.queries.ListComments(ctx, item.id)
	if err != nil {
		return nil, err
	}
	comments := []Comment{}
	for _, row := range rows {
		comments = append(comments, commentFromRow(row))
	}
	return comments, nil
}

func (s *Store) CreateLabel(
	ctx context.Context,
	repo string,
	actor Actor,
	name, color, description string,
) (Label, int64, error) {
	if err := checkName("label", name); err != nil {
		return Label{}, 0, err
	}
	color = strings.ToLower(strings.TrimPrefix(color, "#"))
	if color != "" && !colorPattern.MatchString(color) {
		return Label{}, 0, InvalidError("color must be six hex digits, such as d73a4a")
	}
	var label Label
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		var err error
		label, err = q.CreateLabel(ctx, queries.CreateLabelParams{
			Repo:        repo,
			Name:        name,
			Color:       color,
			Description: description,
		})
		if err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, "label.create", repo, 0, actor, labelCreatedEvent{
			Color:       color,
			Description: description,
			Name:        name,
		})
		return err
	})
	return label, eventID, err
}

func (s *Store) Labels(ctx context.Context, repo string) ([]Label, error) {
	labels, err := s.queries.ListLabels(ctx, repo)
	if labels == nil {
		labels = []Label{}
	}
	return labels, err
}

func (s *Store) DeleteLabel(ctx context.Context, repo string, actor Actor, name string) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		if err := deleted(q.DeleteLabel(ctx, queries.DeleteLabelParams{Repo: repo, Name: name})); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(ctx, q, "label.delete", repo, 0, actor, nameEvent{Name: name})
		return err
	})
	return eventID, err
}
