-- +goose Up
CREATE TABLE runner_tokens (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	hash TEXT NOT NULL UNIQUE,
	prefix TEXT NOT NULL,
	user TEXT NOT NULL,
	created_at DATETIME NOT NULL
);

CREATE TABLE runners (
	id INTEGER PRIMARY KEY,
	uuid TEXT NOT NULL UNIQUE,
	token_hash TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	labels TEXT NOT NULL DEFAULT '[]',
	version TEXT NOT NULL DEFAULT '',
	ephemeral BOOLEAN NOT NULL,
	registration_token_id INTEGER REFERENCES runner_tokens (id) ON DELETE SET NULL,
	user TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	last_seen_at DATETIME
);

-- +goose Down
DROP TABLE runners;
DROP TABLE runner_tokens;
