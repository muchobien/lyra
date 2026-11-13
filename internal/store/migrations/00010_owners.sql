-- +goose Up
CREATE TABLE owners (
	name TEXT PRIMARY KEY,
	description TEXT NOT NULL DEFAULT '',
	user TEXT NOT NULL,
	created_at DATETIME NOT NULL
);

-- +goose Down
DROP TABLE owners;
