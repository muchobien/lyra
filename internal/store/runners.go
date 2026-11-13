package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/muchobien/lyra/internal/store/queries"
)

type RunnerToken struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	User      string    `json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

type Runner struct {
	ID         int64      `json:"id"`
	UUID       string     `json:"uuid"`
	Name       string     `json:"name"`
	Labels     []string   `json:"labels"`
	Version    string     `json:"version"`
	Ephemeral  bool       `json:"ephemeral"`
	User       string     `json:"user"`
	CreatedAt  time.Time  `json:"created_at"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	TokenHash  string     `json:"-"`
}

func runnerTokenFromRow(row queries.RunnerToken) RunnerToken {
	return RunnerToken{ID: row.ID, Name: row.Name, Prefix: row.Prefix, User: row.User, CreatedAt: row.CreatedAt}
}

func runnerFromRow(row queries.Runner) (Runner, error) {
	runner := Runner{
		ID: row.ID, UUID: row.Uuid, Name: row.Name, Version: row.Version, Ephemeral: row.Ephemeral, User: row.User,
		CreatedAt: row.CreatedAt, TokenHash: row.TokenHash,
	}
	if row.LastSeenAt.Valid {
		runner.LastSeenAt = &row.LastSeenAt.Time
	}
	return runner, json.Unmarshal([]byte(row.Labels), &runner.Labels)
}

func (s *Store) CreateRunnerToken(actor Actor, name, secret string) (RunnerToken, int64, error) {
	var created RunnerToken
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		row, err := q.CreateRunnerToken(context.Background(), queries.CreateRunnerTokenParams{
			Name: name, Hash: HashToken(secret), Prefix: secret[:8], User: actor.User, CreatedAt: now(),
		})
		if err != nil {
			return err
		}
		created = runnerTokenFromRow(row)
		eventID, err = addEvent(q, "runner.token.create", "", 0, actor, map[string]any{"id": created.ID, "name": created.Name})
		return err
	})
	return created, eventID, err
}

func (s *Store) RunnerTokenBySecret(secret string) (RunnerToken, error) {
	row, err := s.queries.GetRunnerTokenByHash(context.Background(), HashToken(secret))
	if err != nil {
		return RunnerToken{}, notFound(err)
	}
	return runnerTokenFromRow(row), nil
}

func (s *Store) RunnerTokens() ([]RunnerToken, error) {
	rows, err := s.queries.ListRunnerTokens(context.Background())
	if err != nil {
		return nil, err
	}
	tokens := []RunnerToken{}
	for _, row := range rows {
		tokens = append(tokens, runnerTokenFromRow(row))
	}
	return tokens, nil
}

func (s *Store) DeleteRunnerToken(actor Actor, id int64) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		if err := deleted(q.DeleteRunnerToken(context.Background(), id)); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(q, "runner.token.delete", "", 0, actor, map[string]any{"id": id})
		return err
	})
	return eventID, err
}

func (s *Store) CreateRunner(actor Actor, runner Runner, secret string, registrationToken int64) (Runner, error) {
	var created Runner
	err := s.inTx(func(q *queries.Queries) error {
		row, err := q.CreateRunner(context.Background(), queries.CreateRunnerParams{
			Uuid: runner.UUID, TokenHash: HashToken(secret), Name: runner.Name, Version: runner.Version, Ephemeral: runner.Ephemeral,
			RegistrationTokenID: sql.NullInt64{Int64: registrationToken, Valid: registrationToken != 0}, User: actor.User, CreatedAt: now(),
			LastSeenAt: sql.NullTime{Time: now(), Valid: true},
		})
		if err != nil {
			return err
		}
		if created, err = runnerFromRow(row); err != nil {
			return err
		}
		_, err = addEvent(q, "runner.register", "", 0, actor, map[string]any{
			"id": created.ID, "name": created.Name, "version": created.Version, "ephemeral": created.Ephemeral,
		})
		return err
	})
	return created, err
}

func (s *Store) RunnerByUUID(uuid string) (Runner, error) {
	row, err := s.queries.GetRunnerByUUID(context.Background(), uuid)
	if err != nil {
		return Runner{}, notFound(err)
	}
	return runnerFromRow(row)
}

func (s *Store) DeclareRunner(id int64, labels []string, version string) (Runner, error) {
	encoded, err := json.Marshal(labels)
	if err != nil {
		return Runner{}, err
	}
	row, err := s.queries.DeclareRunner(context.Background(), queries.DeclareRunnerParams{
		Labels: string(encoded), Version: version, LastSeenAt: sql.NullTime{Time: now(), Valid: true}, ID: id,
	})
	if err != nil {
		return Runner{}, notFound(err)
	}
	return runnerFromRow(row)
}

func (s *Store) TouchRunner(id int64) error {
	return s.queries.TouchRunner(context.Background(), queries.TouchRunnerParams{LastSeenAt: sql.NullTime{Time: now(), Valid: true}, ID: id})
}

func (s *Store) Runners() ([]Runner, error) {
	rows, err := s.queries.ListRunners(context.Background())
	if err != nil {
		return nil, err
	}
	runners := []Runner{}
	for _, row := range rows {
		runner, err := runnerFromRow(row)
		if err != nil {
			return nil, err
		}
		runners = append(runners, runner)
	}
	return runners, nil
}

func (s *Store) DeleteRunner(actor Actor, id int64) (int64, error) {
	var eventID int64
	err := s.inTx(func(q *queries.Queries) error {
		if err := deleted(q.DeleteRunner(context.Background(), id)); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(q, "runner.delete", "", 0, actor, map[string]any{"id": id})
		return err
	})
	return eventID, err
}
