package store

import (
	"context"
	"database/sql"
	"encoding/json/v2"
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

type idEvent struct {
	ID int64 `json:"id"`
}

type runnerTokenCreatedEvent struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type runnerRegisteredEvent struct {
	Ephemeral bool   `json:"ephemeral"`
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

func runnerTokenFromRow(row queries.RunnerToken) RunnerToken {
	return RunnerToken{ID: row.ID, Name: row.Name, Prefix: row.Prefix, User: row.User, CreatedAt: row.CreatedAt}
}

func runnerFromRow(row queries.Runner) (Runner, error) {
	runner := Runner{
		ID:        row.ID,
		UUID:      row.Uuid,
		Name:      row.Name,
		Version:   row.Version,
		Ephemeral: row.Ephemeral,
		User:      row.User,
		CreatedAt: row.CreatedAt,
		TokenHash: row.TokenHash,
	}
	if row.LastSeenAt.Valid {
		runner.LastSeenAt = &row.LastSeenAt.Time
	}
	return runner, json.Unmarshal([]byte(row.Labels), &runner.Labels)
}

func (s *Store) CreateRunnerToken(ctx context.Context, actor Actor, name, secret string) (RunnerToken, int64, error) {
	var created RunnerToken
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := q.CreateRunnerToken(ctx, queries.CreateRunnerTokenParams{
			Name:      name,
			Hash:      HashToken(secret),
			Prefix:    secret[:8],
			User:      actor.User,
			CreatedAt: now(),
		})
		if err != nil {
			return err
		}
		created = runnerTokenFromRow(row)
		data := runnerTokenCreatedEvent{ID: created.ID, Name: created.Name}
		eventID, err = addEvent(ctx, q, "runner.token.create", "", 0, actor, data)
		return err
	})
	return created, eventID, err
}

func (s *Store) RunnerTokenBySecret(ctx context.Context, secret string) (RunnerToken, error) {
	row, err := s.queries.GetRunnerTokenByHash(ctx, HashToken(secret))
	if err != nil {
		return RunnerToken{}, notFound(err)
	}
	return runnerTokenFromRow(row), nil
}

func (s *Store) RunnerTokens(ctx context.Context) ([]RunnerToken, error) {
	rows, err := s.queries.ListRunnerTokens(ctx)
	if err != nil {
		return nil, err
	}
	tokens := []RunnerToken{}
	for _, row := range rows {
		tokens = append(tokens, runnerTokenFromRow(row))
	}
	return tokens, nil
}

func (s *Store) DeleteRunnerToken(ctx context.Context, actor Actor, id int64) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		if err := deleted(q.DeleteRunnerToken(ctx, id)); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(ctx, q, "runner.token.delete", "", 0, actor, idEvent{ID: id})
		return err
	})
	return eventID, err
}

func (s *Store) CreateRunner(
	ctx context.Context,
	actor Actor,
	runner Runner,
	secret string,
	registrationToken int64,
) (Runner, error) {
	var created Runner
	err := s.inTx(ctx, func(q *queries.Queries) error {
		row, err := q.CreateRunner(ctx, queries.CreateRunnerParams{
			Uuid:                runner.UUID,
			TokenHash:           HashToken(secret),
			Name:                runner.Name,
			Version:             runner.Version,
			Ephemeral:           runner.Ephemeral,
			RegistrationTokenID: sql.NullInt64{Int64: registrationToken, Valid: registrationToken != 0},
			User:                actor.User,
			CreatedAt:           now(),
			LastSeenAt:          sql.NullTime{Time: now(), Valid: true},
		})
		if err != nil {
			return err
		}
		if created, err = runnerFromRow(row); err != nil {
			return err
		}
		_, err = addEvent(ctx, q, "runner.register", "", 0, actor, runnerRegisteredEvent{
			Ephemeral: created.Ephemeral,
			ID:        created.ID,
			Name:      created.Name,
			Version:   created.Version,
		})
		return err
	})
	return created, err
}

func (s *Store) RunnerByUUID(ctx context.Context, uuid string) (Runner, error) {
	row, err := s.queries.GetRunnerByUUID(ctx, uuid)
	if err != nil {
		return Runner{}, notFound(err)
	}
	return runnerFromRow(row)
}

func (s *Store) DeclareRunner(ctx context.Context, id int64, labels []string, version string) (Runner, error) {
	encoded, err := json.Marshal(labels, storedJSON)
	if err != nil {
		return Runner{}, err
	}
	row, err := s.queries.DeclareRunner(ctx, queries.DeclareRunnerParams{
		Labels:     string(encoded),
		Version:    version,
		LastSeenAt: sql.NullTime{Time: now(), Valid: true},
		ID:         id,
	})
	if err != nil {
		return Runner{}, notFound(err)
	}
	return runnerFromRow(row)
}

func (s *Store) TouchRunner(ctx context.Context, id int64) error {
	return s.queries.TouchRunner(ctx, queries.TouchRunnerParams{
		LastSeenAt: sql.NullTime{Time: now(), Valid: true},
		ID:         id,
	})
}

func (s *Store) Runners(ctx context.Context) ([]Runner, error) {
	rows, err := s.queries.ListRunners(ctx)
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

func (s *Store) DeleteRunner(ctx context.Context, actor Actor, id int64) (int64, error) {
	var eventID int64
	err := s.inTx(ctx, func(q *queries.Queries) error {
		if err := deleted(q.DeleteRunner(ctx, id)); err != nil {
			return err
		}
		var err error
		eventID, err = addEvent(ctx, q, "runner.delete", "", 0, actor, idEvent{ID: id})
		return err
	})
	return eventID, err
}
