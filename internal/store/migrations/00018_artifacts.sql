-- +goose Up
CREATE TABLE artifacts (
	id INTEGER PRIMARY KEY,
	run_id INTEGER NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
	job_id INTEGER NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	object TEXT NOT NULL,
	size INTEGER NOT NULL DEFAULT 0,
	sha256 TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	expires_at DATETIME,
	UNIQUE (run_id, name)
);

-- +goose Down
DROP TABLE artifacts;
