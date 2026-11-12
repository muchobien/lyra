-- +goose Up
CREATE TABLE reviews (
	id INTEGER PRIMARY KEY,
	item_id INTEGER NOT NULL REFERENCES items (id) ON DELETE CASCADE,
	state TEXT NOT NULL,
	body TEXT NOT NULL DEFAULT '',
	commit_sha TEXT NOT NULL,
	user TEXT NOT NULL,
	via TEXT NOT NULL,
	agent TEXT NOT NULL DEFAULT '',
	session TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL
);

CREATE INDEX reviews_item ON reviews (item_id, id);

ALTER TABLE comments ADD COLUMN review_id INTEGER REFERENCES reviews (id) ON DELETE CASCADE;
ALTER TABLE comments ADD COLUMN commit_sha TEXT NOT NULL DEFAULT '';
ALTER TABLE comments ADD COLUMN path TEXT NOT NULL DEFAULT '';
ALTER TABLE comments ADD COLUMN side TEXT NOT NULL DEFAULT '';
ALTER TABLE comments ADD COLUMN line INTEGER NOT NULL DEFAULT 0;

ALTER TABLE branch_rules ADD COLUMN require_approval BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE branch_rules DROP COLUMN require_approval;
ALTER TABLE comments DROP COLUMN line;
ALTER TABLE comments DROP COLUMN side;
ALTER TABLE comments DROP COLUMN path;
ALTER TABLE comments DROP COLUMN commit_sha;
ALTER TABLE comments DROP COLUMN review_id;
DROP TABLE reviews;
