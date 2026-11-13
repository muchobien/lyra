package store

import (
	"context"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

type Review struct {
	ID        int64     `json:"id"`
	State     string    `json:"state"`
	Body      string    `json:"body"`
	CommitSHA string    `json:"commit_sha"`
	Author    Actor     `json:"author"`
	Comments  []Comment `json:"comments"`
	CreatedAt time.Time `json:"created_at"`
}

type LineComment struct {
	LineRef
	Body string `json:"body"`
}

type reviewEvent struct {
	Body      string    `json:"body"`
	Comments  []Comment `json:"comments"`
	CommitSHA string    `json:"commit_sha"`
	Number    int64     `json:"number"`
	Review    int64     `json:"review"`
	State     string    `json:"state"`
}

func reviewFromRow(row queries.Review) Review {
	return Review{
		ID:        row.ID,
		State:     row.State,
		Body:      row.Body,
		CommitSHA: row.CommitSha,
		Comments:  []Comment{},
		Author:    Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
		CreatedAt: row.CreatedAt,
	}
}

func (s *Store) CreateReview(
	ctx context.Context,
	repo string,
	number int64,
	actor Actor,
	state, body, commitSHA string,
	comments []LineComment,
) (Review, int64, error) {
	var review Review
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := getPull(ctx, q, repo, number)
		if err != nil {
			return err
		}
		at := now()
		created, err := q.CreateReview(ctx, queries.CreateReviewParams{
			ItemID:    row.Item.ID,
			State:     state,
			Body:      body,
			CommitSha: commitSHA,
			User:      actor.User,
			Via:       actor.Via,
			Agent:     actor.Agent,
			Session:   actor.Session,
			CreatedAt: at,
		})
		if err != nil {
			return err
		}
		review = reviewFromRow(created)
		for _, comment := range comments {
			line := comment.LineRef
			line.CommitSHA = commitSHA
			saved, err := createComment(ctx, q, row.Item.ID, actor, comment.Body, created.ID, line, at)
			if err != nil {
				return err
			}
			review.Comments = append(review.Comments, commentFromRow(saved))
		}
		if err := q.TouchItem(ctx, queries.TouchItemParams{UpdatedAt: at, ID: row.Item.ID}); err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, "pr.review", repo, number, actor, reviewEvent{
			Body:      body,
			Comments:  review.Comments,
			CommitSHA: commitSHA,
			Number:    number,
			Review:    created.ID,
			State:     state,
		})
		return err
	})
	return review, eventID, err
}

func (s *Store) Reviews(ctx context.Context, item Item) ([]Review, error) {
	rows, err := s.queries.ListReviews(ctx, item.id)
	if err != nil {
		return nil, err
	}
	comments, err := s.queries.ListComments(ctx, item.id)
	if err != nil {
		return nil, err
	}
	reviews := []Review{}
	for _, row := range rows {
		review := reviewFromRow(row)
		for _, comment := range comments {
			if comment.ReviewID.Int64 == row.ID {
				review.Comments = append(review.Comments, commentFromRow(comment))
			}
		}
		reviews = append(reviews, review)
	}
	return reviews, nil
}
