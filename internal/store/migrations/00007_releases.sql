-- +goose Up
CREATE TABLE releases (
	id INTEGER PRIMARY KEY,
	repo TEXT NOT NULL REFERENCES repos (name) ON DELETE CASCADE,
	tag TEXT NOT NULL,
	title TEXT NOT NULL,
	notes TEXT NOT NULL DEFAULT '',
	draft BOOLEAN NOT NULL,
	prerelease BOOLEAN NOT NULL,
	target TEXT NOT NULL,
	commit_sha TEXT NOT NULL DEFAULT '',
	user TEXT NOT NULL,
	via TEXT NOT NULL,
	agent TEXT NOT NULL DEFAULT '',
	session TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	published_at DATETIME,
	UNIQUE (repo, tag)
);

CREATE TABLE release_assets (
	id INTEGER PRIMARY KEY,
	release_id INTEGER NOT NULL REFERENCES releases (id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	object TEXT NOT NULL,
	size INTEGER NOT NULL,
	sha256 TEXT NOT NULL,
	content_type TEXT NOT NULL,
	user TEXT NOT NULL,
	via TEXT NOT NULL,
	agent TEXT NOT NULL DEFAULT '',
	session TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	UNIQUE (release_id, name)
);

-- +goose Down
DROP TABLE release_assets;
DROP TABLE releases;
