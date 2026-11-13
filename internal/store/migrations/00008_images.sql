-- +goose Up
CREATE TABLE images (
	id INTEGER PRIMARY KEY,
	repo TEXT NOT NULL REFERENCES repos (name) ON DELETE CASCADE,
	name TEXT NOT NULL,
	tag TEXT NOT NULL,
	digest TEXT NOT NULL,
	media_type TEXT NOT NULL,
	user TEXT NOT NULL,
	via TEXT NOT NULL,
	agent TEXT NOT NULL DEFAULT '',
	session TEXT NOT NULL DEFAULT '',
	pushed_at DATETIME NOT NULL,
	UNIQUE (name, tag)
);

-- +goose Down
DROP TABLE images;
