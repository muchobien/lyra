-- +goose Up
CREATE TABLE secrets (
	repo TEXT NOT NULL REFERENCES repos (name) ON DELETE CASCADE,
	name TEXT NOT NULL,
	value BLOB NOT NULL,
	user TEXT NOT NULL,
	updated_at DATETIME NOT NULL,
	PRIMARY KEY (repo, name)
);

CREATE TABLE variables (
	repo TEXT NOT NULL REFERENCES repos (name) ON DELETE CASCADE,
	name TEXT NOT NULL,
	value TEXT NOT NULL,
	user TEXT NOT NULL,
	updated_at DATETIME NOT NULL,
	PRIMARY KEY (repo, name)
);

-- +goose Down
DROP TABLE variables;
DROP TABLE secrets;
