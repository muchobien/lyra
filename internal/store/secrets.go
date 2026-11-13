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

func (s *Store) SetSecret(actor Actor, repo, name string, value []byte) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		if err := q.SetSecret(context.Background(), queries.SetSecretParams{Repo: repo, Name: name, Value: value, User: actor.User, UpdatedAt: now()}); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(q, "secret.set", repo, 0, actor, map[string]any{"name": name})
		return err
	})
	return eventID, err
}

func (s *Store) Secrets(repo string) ([]Secret, error) {
	rows, err := s.queries.ListSecrets(context.Background(), repo)
	if err != nil {
		return nil, err
	}
	secrets := []Secret{}
	for _, row := range rows {
		secrets = append(secrets, Secret{Name: row.Name, User: row.User, UpdatedAt: row.UpdatedAt, Value: row.Value})
	}
	return secrets, nil
}

func (s *Store) DeleteSecret(actor Actor, repo, name string) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		if err := deleted(q.DeleteSecret(context.Background(), queries.DeleteSecretParams{Repo: repo, Name: name})); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(q, "secret.delete", repo, 0, actor, map[string]any{"name": name})
		return err
	})
	return eventID, err
}

func (s *Store) SetVariable(actor Actor, repo, name, value string) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		if err := q.SetVariable(context.Background(), queries.SetVariableParams{Repo: repo, Name: name, Value: value, User: actor.User, UpdatedAt: now()}); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(q, "variable.set", repo, 0, actor, map[string]any{"name": name})
		return err
	})
	return eventID, err
}

func (s *Store) Variables(repo string) ([]Variable, error) {
	rows, err := s.queries.ListVariables(context.Background(), repo)
	if err != nil {
		return nil, err
	}
	variables := []Variable{}
	for _, row := range rows {
		variables = append(variables, Variable{Name: row.Name, Value: row.Value, User: row.User, UpdatedAt: row.UpdatedAt})
	}
	return variables, nil
}

func (s *Store) DeleteVariable(actor Actor, repo, name string) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		if err := deleted(q.DeleteVariable(context.Background(), queries.DeleteVariableParams{Repo: repo, Name: name})); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(q, "variable.delete", repo, 0, actor, map[string]any{"name": name})
		return err
	})
	return eventID, err
}
