package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"github.com/muchobien/lyra/internal/store/queries"
)

//go:embed migrations/*.sql
var migrations embed.FS

var ErrNotFound = errors.New("not found")

type Store struct {
	db      *sql.DB
	queries *queries.Queries
	mutex   sync.Mutex
	changed chan struct{}
}

type Actor struct {
	User    string `json:"user"`
	Via     string `json:"via"`
	Agent   string `json:"agent,omitempty"`
	Session string `json:"session,omitempty"`
}

type Repo = queries.Repo

type Key = queries.Key

type Token struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	User       string     `json:"user"`
	Prefix     string     `json:"prefix"`
	Scope      string     `json:"scope"`
	Repos      []string   `json:"repos"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

type Event struct {
	ID     int64           `json:"id"`
	At     time.Time       `json:"at"`
	Action string          `json:"action"`
	Repo   string          `json:"repo,omitempty"`
	Actor  Actor           `json:"actor"`
	Data   json.RawMessage `json:"data"`
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_time_format=sqlite")
	if err != nil {
		return nil, err
	}
	return &Store{db: db, queries: queries.New(db), changed: make(chan struct{})}, nil
}

func (s *Store) Migrate(ctx context.Context) error {
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, s.db, files)
	if err != nil {
		return err
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	for _, result := range results {
		slog.Info("migrated", "version", result.Source.Version, "file", result.Source.Path, "duration", result.Duration)
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func now() time.Time {
	return time.Now().UTC()
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func deleted(count int64, err error) error {
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func HashToken(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func (s *Store) Changed() <-chan struct{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.changed
}

func (s *Store) Notify() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	close(s.changed)
	s.changed = make(chan struct{})
}

func (s *Store) AddEvent(action, repo string, actor Actor, data any) (int64, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	id, err := s.queries.CreateEvent(context.Background(), queries.CreateEventParams{
		At: now(), Action: action, Repo: repo, User: actor.User, Via: actor.Via, Agent: actor.Agent, Session: actor.Session, Data: string(encoded),
	})
	if err != nil {
		return 0, err
	}
	s.Notify()
	return id, nil
}

func eventsFromRows(rows []queries.Event) []Event {
	events := []Event{}
	for _, row := range rows {
		events = append(events, Event{
			ID:     row.ID,
			At:     row.At,
			Action: row.Action,
			Repo:   row.Repo,
			Actor:  Actor{User: row.User, Via: row.Via, Agent: row.Agent, Session: row.Session},
			Data:   json.RawMessage(row.Data),
		})
	}
	return events
}

func (s *Store) EventsAfter(id int64, limit int) ([]Event, error) {
	rows, err := s.queries.ListEventsAfter(context.Background(), queries.ListEventsAfterParams{ID: id, Limit: int64(limit)})
	if err != nil {
		return nil, err
	}
	return eventsFromRows(rows), nil
}

func (s *Store) LatestEventID() (int64, error) {
	return s.queries.LatestEventID(context.Background())
}

func (s *Store) Events(repo string, limit int) ([]Event, error) {
	var rows []queries.Event
	var err error
	if repo == "" {
		rows, err = s.queries.ListEvents(context.Background(), int64(limit))
	} else {
		rows, err = s.queries.ListRepoEvents(context.Background(), queries.ListRepoEventsParams{Repo: repo, Limit: int64(limit)})
	}
	if err != nil {
		return nil, err
	}
	return eventsFromRows(rows), nil
}

func (s *Store) CreateRepo(name, description, createdBy string) (Repo, error) {
	return s.queries.CreateRepo(context.Background(), queries.CreateRepoParams{Name: name, Description: description, CreatedAt: now(), CreatedBy: createdBy})
}

func (s *Store) Repo(name string) (Repo, error) {
	repo, err := s.queries.GetRepo(context.Background(), name)
	return repo, notFound(err)
}

func (s *Store) Repos() ([]Repo, error) {
	repos, err := s.queries.ListRepos(context.Background())
	if repos == nil {
		repos = []Repo{}
	}
	return repos, err
}

func (s *Store) DeleteRepo(name string) error {
	return deleted(s.queries.DeleteRepo(context.Background(), name))
}

func tokenFromRow(row queries.Token) (Token, error) {
	token := Token{ID: row.ID, Name: row.Name, User: row.User, Prefix: row.Prefix, Scope: row.Scope, CreatedAt: row.CreatedAt}
	if err := json.Unmarshal([]byte(row.Repos), &token.Repos); err != nil {
		return token, err
	}
	if row.ExpiresAt.Valid {
		token.ExpiresAt = &row.ExpiresAt.Time
	}
	if row.LastUsedAt.Valid {
		token.LastUsedAt = &row.LastUsedAt.Time
	}
	return token, nil
}

func (s *Store) CreateToken(name, user, secret, scope string, repos []string, expiresAt *time.Time) (Token, error) {
	encodedRepos, err := json.Marshal(repos)
	if err != nil {
		return Token{}, err
	}
	expires := sql.NullTime{}
	if expiresAt != nil {
		expires = sql.NullTime{Time: expiresAt.UTC(), Valid: true}
	}
	row, err := s.queries.CreateToken(context.Background(), queries.CreateTokenParams{
		Name: name, User: user, Hash: HashToken(secret), Prefix: secret[:12], Scope: scope, Repos: string(encodedRepos), CreatedAt: now(), ExpiresAt: expires,
	})
	if err != nil {
		return Token{}, err
	}
	return tokenFromRow(row)
}

func (s *Store) TokenBySecret(secret string) (Token, error) {
	row, err := s.queries.UseToken(context.Background(), queries.UseTokenParams{LastUsedAt: sql.NullTime{Time: now(), Valid: true}, Hash: HashToken(secret)})
	if err != nil {
		return Token{}, notFound(err)
	}
	return tokenFromRow(row)
}

func (s *Store) Tokens(user string) ([]Token, error) {
	rows, err := s.queries.ListTokens(context.Background(), user)
	if err != nil {
		return nil, err
	}
	tokens := []Token{}
	for _, row := range rows {
		token, err := tokenFromRow(row)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (s *Store) TokenActive(id int64) (bool, error) {
	count, err := s.queries.TokenActive(context.Background(), queries.TokenActiveParams{ID: id, ExpiresAt: sql.NullTime{Time: now(), Valid: true}})
	return count > 0, err
}

func (s *Store) DeleteToken(user string, id int64) error {
	return deleted(s.queries.DeleteToken(context.Background(), queries.DeleteTokenParams{User: user, ID: id}))
}

func (s *Store) CreateKey(key Key) (Key, error) {
	return s.queries.CreateKey(context.Background(), queries.CreateKeyParams{
		User: key.User, Title: key.Title, Agent: key.Agent, PublicKey: key.PublicKey, Fingerprint: key.Fingerprint, Signing: key.Signing, CreatedAt: now(),
	})
}

func (s *Store) Keys(user string) ([]Key, error) {
	var keys []Key
	var err error
	if user == "" {
		keys, err = s.queries.ListKeys(context.Background())
	} else {
		keys, err = s.queries.ListUserKeys(context.Background(), user)
	}
	if keys == nil {
		keys = []Key{}
	}
	return keys, err
}

func (s *Store) DeleteKey(user string, id int64) error {
	return deleted(s.queries.DeleteKey(context.Background(), queries.DeleteKeyParams{User: user, ID: id}))
}

func IsUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
