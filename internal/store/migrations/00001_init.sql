-- +goose Up
CREATE TABLE repos (
	name TEXT PRIMARY KEY,
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	created_by TEXT NOT NULL
);

CREATE TABLE tokens (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	user TEXT NOT NULL,
	hash TEXT NOT NULL UNIQUE,
	prefix TEXT NOT NULL,
	scope TEXT NOT NULL,
	repos TEXT NOT NULL DEFAULT '[]',
	created_at TEXT NOT NULL,
	expires_at TEXT,
	last_used_at TEXT
);

CREATE TABLE keys (
	id INTEGER PRIMARY KEY,
	user TEXT NOT NULL,
	title TEXT NOT NULL,
	agent TEXT NOT NULL DEFAULT '',
	public_key TEXT NOT NULL UNIQUE,
	fingerprint TEXT NOT NULL UNIQUE,
	signing INTEGER NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE events (
	id INTEGER PRIMARY KEY,
	at TEXT NOT NULL,
	action TEXT NOT NULL,
	repo TEXT NOT NULL DEFAULT '',
	user TEXT NOT NULL,
	via TEXT NOT NULL,
	agent TEXT NOT NULL DEFAULT '',
	session TEXT NOT NULL DEFAULT '',
	data TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX events_repo ON events (repo, id);

-- +goose Down
DROP TABLE events;
DROP TABLE keys;
DROP TABLE tokens;
DROP TABLE repos;
