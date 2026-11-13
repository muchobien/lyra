package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/muchobien/lyra/internal/store/queries"
)

type Owner = queries.Owner

func (s *Store) CreateOwner(actor Actor, name, description string) (Owner, int64, error) {
	var owner Owner
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		var err error
		owner, err = q.CreateOwner(context.Background(), queries.CreateOwnerParams{Name: name, Description: description, User: actor.User, CreatedAt: now()})
		if IsUniqueViolation(err) {
			return ErrExists
		}
		if err != nil {
			return err
		}
		eventID, err = addEvent(q, "owner.create", "", 0, actor, map[string]any{"name": name, "description": description})
		return err
	})
	return owner, eventID, err
}

func (s *Store) Owner(name string) (Owner, error) {
	owner, err := s.queries.GetOwner(context.Background(), name)
	return owner, notFound(err)
}

func (s *Store) Owners() ([]Owner, error) {
	return s.queries.ListOwners(context.Background())
}

func (s *Store) DeleteOwner(actor Actor, name string) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		ctx := context.Background()
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
		eventID, err = addEvent(q, "owner.delete", "", 0, actor, map[string]any{"name": name})
		return err
	})
	return eventID, err
}
