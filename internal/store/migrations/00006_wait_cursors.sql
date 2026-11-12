-- +goose Up
CREATE TABLE wait_cursors (
	repo TEXT NOT NULL REFERENCES repos (name) ON DELETE CASCADE,
	item INTEGER NOT NULL,
	user TEXT NOT NULL,
	via TEXT NOT NULL,
	agent TEXT NOT NULL,
	session TEXT NOT NULL,
	event_id INTEGER NOT NULL,
	PRIMARY KEY (repo, item, user, via, agent, session)
);

-- +goose Down
DROP TABLE wait_cursors;
