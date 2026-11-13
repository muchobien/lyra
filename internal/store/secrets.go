package store

import (
	"context"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

type Secret struct {
	Name      string    `json:"name"`
	User      string    `json:"user"`
	UpdatedAt time.Time `json:"updated_at"`
	Value     []byte    `json:"-"`
}

type Variable struct {
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	User      string    `json:"user"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Store) SetSecret(ctx context.Context, actor Actor, repo, name string, value []byte) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		err := q.SetSecret(ctx, queries.SetSecretParams{
			Repo:      repo,
			Name:      name,
			Value:     value,
			User:      actor.User,
			UpdatedAt: now(),
		})
		if err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, "secret.set", repo, 0, actor, nameEvent{Name: name})
		return err
	})
	return eventID, err
}

func (s *Store) Secrets(ctx context.Context, repo string) ([]Secret, error) {
	rows, err := s.queries.ListSecrets(ctx, repo)
	if err != nil {
		return nil, err
	}
	secrets := []Secret{}
	for _, row := range rows {
		secrets = append(secrets, Secret{Name: row.Name, User: row.User, UpdatedAt: row.UpdatedAt, Value: row.Value})
	}
	return secrets, nil
}

func (s *Store) DeleteSecret(ctx context.Context, actor Actor, repo, name string) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		if err := deleted(q.DeleteSecret(ctx, queries.DeleteSecretParams{Repo: repo, Name: name})); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(ctx, q, "secret.delete", repo, 0, actor, nameEvent{Name: name})
		return err
	})
	return eventID, err
}

func (s *Store) SetVariable(ctx context.Context, actor Actor, repo, name, value string) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		err := q.SetVariable(ctx, queries.SetVariableParams{
			Repo:      repo,
			Name:      name,
			Value:     value,
			User:      actor.User,
			UpdatedAt: now(),
		})
		if err != nil {
			return err
		}
		eventID, err = addEvent(ctx, q, "variable.set", repo, 0, actor, nameEvent{Name: name})
		return err
	})
	return eventID, err
}

func (s *Store) Variables(ctx context.Context, repo string) ([]Variable, error) {
	rows, err := s.queries.ListVariables(ctx, repo)
	if err != nil {
		return nil, err
	}
	variables := []Variable{}
	for _, row := range rows {
		variables = append(variables, Variable{
			Name:      row.Name,
			Value:     row.Value,
			User:      row.User,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return variables, nil
}

func (s *Store) DeleteVariable(ctx context.Context, actor Actor, repo, name string) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		if err := deleted(q.DeleteVariable(ctx, queries.DeleteVariableParams{Repo: repo, Name: name})); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(ctx, q, "variable.delete", repo, 0, actor, nameEvent{Name: name})
		return err
	})
	return eventID, err
}
