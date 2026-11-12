-- +goose Up
CREATE TABLE items (
	id INTEGER PRIMARY KEY,
	repo TEXT NOT NULL REFERENCES repos (name) ON DELETE CASCADE,
	number INTEGER NOT NULL,
	kind TEXT NOT NULL,
	title TEXT NOT NULL,
	body TEXT NOT NULL DEFAULT '',
	state TEXT NOT NULL,
	user TEXT NOT NULL,
	via TEXT NOT NULL,
	agent TEXT NOT NULL DEFAULT '',
	session TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	closed_at DATETIME,
	UNIQUE (repo, number)
);

CREATE TABLE labels (
	id INTEGER PRIMARY KEY,
	repo TEXT NOT NULL REFERENCES repos (name) ON DELETE CASCADE,
	name TEXT NOT NULL,
	color TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	UNIQUE (repo, name)
);

CREATE TABLE item_labels (
	item_id INTEGER NOT NULL REFERENCES items (id) ON DELETE CASCADE,
	label_id INTEGER NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
	PRIMARY KEY (item_id, label_id)
);

CREATE TABLE item_assignees (
	item_id INTEGER NOT NULL REFERENCES items (id) ON DELETE CASCADE,
	assignee TEXT NOT NULL,
	PRIMARY KEY (item_id, assignee)
);

CREATE TABLE comments (
	id INTEGER PRIMARY KEY,
	item_id INTEGER NOT NULL REFERENCES items (id) ON DELETE CASCADE,
	body TEXT NOT NULL,
	user TEXT NOT NULL,
	via TEXT NOT NULL,
	agent TEXT NOT NULL DEFAULT '',
	session TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL
);

CREATE INDEX comments_item ON comments (item_id, id);

ALTER TABLE events ADD COLUMN item INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE events DROP COLUMN item;
DROP TABLE comments;
DROP TABLE item_assignees;
DROP TABLE item_labels;
DROP TABLE labels;
DROP TABLE items;
