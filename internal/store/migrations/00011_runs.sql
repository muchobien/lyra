-- +goose Up
CREATE TABLE runs (
	id INTEGER PRIMARY KEY,
	repo TEXT NOT NULL REFERENCES repos (name) ON DELETE CASCADE,
	number INTEGER NOT NULL,
	workflow TEXT NOT NULL,
	name TEXT NOT NULL,
	title TEXT NOT NULL,
	event TEXT NOT NULL,
	ref TEXT NOT NULL,
	sha TEXT NOT NULL,
	payload TEXT NOT NULL,
	status TEXT NOT NULL,
	user TEXT NOT NULL,
	via TEXT NOT NULL,
	agent TEXT NOT NULL DEFAULT '',
	session TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	started_at DATETIME,
	stopped_at DATETIME,
	UNIQUE (repo, number)
);

CREATE TABLE jobs (
	id INTEGER PRIMARY KEY,
	run_id INTEGER NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
	job_key TEXT NOT NULL,
	name TEXT NOT NULL,
	needs TEXT NOT NULL,
	runs_on TEXT NOT NULL,
	conditional BOOLEAN NOT NULL,
	payload TEXT NOT NULL,
	status TEXT NOT NULL,
	attempt INTEGER NOT NULL DEFAULT 0,
	outputs TEXT NOT NULL DEFAULT '{}',
	started_at DATETIME,
	stopped_at DATETIME
);

CREATE INDEX jobs_waiting ON jobs (status, id);

CREATE TABLE tasks (
	id INTEGER PRIMARY KEY,
	job_id INTEGER NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
	runner_id INTEGER REFERENCES runners (id) ON DELETE SET NULL,
	attempt INTEGER NOT NULL,
	token_hash TEXT NOT NULL UNIQUE,
	status TEXT NOT NULL,
	steps TEXT NOT NULL DEFAULT '[]',
	log_length INTEGER NOT NULL DEFAULT 0,
	log_object TEXT NOT NULL DEFAULT '',
	started_at DATETIME NOT NULL,
	stopped_at DATETIME
);

CREATE TABLE log_rows (
	task_id INTEGER NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
	idx INTEGER NOT NULL,
	at DATETIME NOT NULL,
	content TEXT NOT NULL,
	PRIMARY KEY (task_id, idx)
);

-- +goose Down
DROP TABLE log_rows;
DROP TABLE tasks;
DROP TABLE jobs;
DROP TABLE runs;
