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
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
}

type Actor struct {
	User    string `json:"user"`
	Via     string `json:"via"`
	Agent   string `json:"agent,omitempty"`
	Session string `json:"session,omitempty"`
}

type Repo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
}

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

type Key struct {
	ID          int64     `json:"id"`
	User        string    `json:"user"`
	Title       string    `json:"title"`
	Agent       string    `json:"agent,omitempty"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	Signing     bool      `json:"signing"`
	CreatedAt   time.Time `json:"created_at"`
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
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
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

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}

func parseNullTime(value sql.NullString) *time.Time {
	if !value.Valid {
		return nil
	}
	parsed := parseTime(value.String)
	return &parsed
}

func HashToken(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func (s *Store) AddEvent(action, repo string, actor Actor, data any) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO events (at, action, repo, user, via, agent, session, data) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		now(), action, repo, actor.User, actor.Via, actor.Agent, actor.Session, string(encoded))
	return err
}

func (s *Store) Events(repo string, limit int) ([]Event, error) {
	query := `SELECT id, at, action, repo, user, via, agent, session, data FROM events`
	args := []any{}
	if repo != "" {
		query += ` WHERE repo = ?`
		args = append(args, repo)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var event Event
		var at, data string
		if err := rows.Scan(&event.ID, &at, &event.Action, &event.Repo, &event.Actor.User, &event.Actor.Via, &event.Actor.Agent, &event.Actor.Session, &data); err != nil {
			return nil, err
		}
		event.At = parseTime(at)
		event.Data = json.RawMessage(data)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) CreateRepo(name, description, createdBy string) (Repo, error) {
	repo := Repo{Name: name, Description: description, CreatedBy: createdBy}
	at := now()
	if _, err := s.db.Exec(`INSERT INTO repos (name, description, created_at, created_by) VALUES (?, ?, ?, ?)`, name, description, at, createdBy); err != nil {
		return repo, err
	}
	repo.CreatedAt = parseTime(at)
	return repo, nil
}

func (s *Store) Repo(name string) (Repo, error) {
	var repo Repo
	var at string
	err := s.db.QueryRow(`SELECT name, description, created_at, created_by FROM repos WHERE name = ?`, name).Scan(&repo.Name, &repo.Description, &at, &repo.CreatedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return repo, ErrNotFound
	}
	repo.CreatedAt = parseTime(at)
	return repo, err
}

func (s *Store) Repos() ([]Repo, error) {
	rows, err := s.db.Query(`SELECT name, description, created_at, created_by FROM repos ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	repos := []Repo{}
	for rows.Next() {
		var repo Repo
		var at string
		if err := rows.Scan(&repo.Name, &repo.Description, &at, &repo.CreatedBy); err != nil {
			return nil, err
		}
		repo.CreatedAt = parseTime(at)
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}

func (s *Store) DeleteRepo(name string) error {
	result, err := s.db.Exec(`DELETE FROM repos WHERE name = ?`, name)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	return nil
}

const tokenColumns = `id, name, user, prefix, scope, repos, created_at, expires_at, last_used_at`

func scanToken(row interface{ Scan(...any) error }) (Token, error) {
	var token Token
	var repos, at string
	var expires, used sql.NullString
	if err := row.Scan(&token.ID, &token.Name, &token.User, &token.Prefix, &token.Scope, &repos, &at, &expires, &used); err != nil {
		return token, err
	}
	if err := json.Unmarshal([]byte(repos), &token.Repos); err != nil {
		return token, err
	}
	token.CreatedAt = parseTime(at)
	token.ExpiresAt = parseNullTime(expires)
	token.LastUsedAt = parseNullTime(used)
	return token, nil
}

func (s *Store) CreateToken(name, user, secret, scope string, repos []string, expiresAt *time.Time) (Token, error) {
	encodedRepos, err := json.Marshal(repos)
	if err != nil {
		return Token{}, err
	}
	var expires sql.NullString
	if expiresAt != nil {
		expires = sql.NullString{String: expiresAt.UTC().Format(time.RFC3339Nano), Valid: true}
	}
	row := s.db.QueryRow(`INSERT INTO tokens (name, user, hash, prefix, scope, repos, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING `+tokenColumns,
		name, user, HashToken(secret), secret[:12], scope, string(encodedRepos), now(), expires)
	return scanToken(row)
}

func (s *Store) TokenBySecret(secret string) (Token, error) {
	token, err := scanToken(s.db.QueryRow(`UPDATE tokens SET last_used_at = ? WHERE hash = ? RETURNING `+tokenColumns, now(), HashToken(secret)))
	if errors.Is(err, sql.ErrNoRows) {
		return token, ErrNotFound
	}
	return token, err
}

func (s *Store) Tokens(user string) ([]Token, error) {
	rows, err := s.db.Query(`SELECT `+tokenColumns+` FROM tokens WHERE user = ? ORDER BY id`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tokens := []Token{}
	for rows.Next() {
		token, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (s *Store) DeleteToken(user string, id int64) error {
	result, err := s.db.Exec(`DELETE FROM tokens WHERE user = ? AND id = ?`, user, id)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	return nil
}

const keyColumns = `id, user, title, agent, public_key, fingerprint, signing, created_at`

func scanKey(row interface{ Scan(...any) error }) (Key, error) {
	var key Key
	var at string
	if err := row.Scan(&key.ID, &key.User, &key.Title, &key.Agent, &key.PublicKey, &key.Fingerprint, &key.Signing, &at); err != nil {
		return key, err
	}
	key.CreatedAt = parseTime(at)
	return key, nil
}

func (s *Store) CreateKey(key Key) (Key, error) {
	return scanKey(s.db.QueryRow(`INSERT INTO keys (user, title, agent, public_key, fingerprint, signing, created_at) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING `+keyColumns,
		key.User, key.Title, key.Agent, key.PublicKey, key.Fingerprint, key.Signing, now()))
}

func (s *Store) Keys(user string) ([]Key, error) {
	query := `SELECT ` + keyColumns + ` FROM keys`
	args := []any{}
	if user != "" {
		query += ` WHERE user = ?`
		args = append(args, user)
	}
	rows, err := s.db.Query(query+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []Key{}
	for rows.Next() {
		key, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (s *Store) DeleteKey(user string, id int64) error {
	result, err := s.db.Exec(`DELETE FROM keys WHERE user = ? AND id = ?`, user, id)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return ErrNotFound
	}
	return nil
}

func IsUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
