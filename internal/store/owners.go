package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/muchobien/lyra/internal/store/queries"
)

type Owner = queries.Owner

type ownerCreatedEvent struct {
	Description string `json:"description"`
	Name        string `json:"name"`
}

func (s *Store) CreateOwner(ctx context.Context, actor Actor, name, description string) (Owner, int64, error) {
	var owner Owner
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		var err error
		owner, err = q.CreateOwner(ctx, queries.CreateOwnerParams{
			Name:        name,
			Description: description,
			User:        actor.User,
			CreatedAt:   now(),
		})
		if IsUniqueViolation(err) {
			return ErrExists
		}
		if err != nil {
			return err
		}
		data := ownerCreatedEvent{Description: description, Name: name}
		eventID, err = addEvent(ctx, q, "owner.create", "", 0, actor, data)
		return err
	})
	return owner, eventID, err
}

func (s *Store) Owner(ctx context.Context, name string) (Owner, error) {
	owner, err := s.queries.GetOwner(ctx, name)
	return owner, notFound(err)
}

func (s *Store) Owners(ctx context.Context) ([]Owner, error) {
	return s.queries.ListOwners(ctx)
}

func (s *Store) DeleteOwner(ctx context.Context, actor Actor, name string) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		repos, err := q.ListRepos(ctx)
		if err != nil {
			return err
		}
		for _, repo := range repos {
			if strings.HasPrefix(repo.Name, name+"/") {
				return InvalidError(fmt.Sprintf("owner %s still has repositories, such as %s", name, repo.Name))
			}
		}
		if err := deleted(q.DeleteOwner(ctx, name)); err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, "owner.delete", "", 0, actor, nameEvent{Name: name})
		return err
	})
	return eventID, err
}
