-- +goose Up
CREATE TABLE branch_rules (
	id INTEGER PRIMARY KEY,
	repo TEXT NOT NULL REFERENCES repos (name) ON DELETE CASCADE,
	pattern TEXT NOT NULL,
	require_signed BOOLEAN NOT NULL DEFAULT 0,
	merge_methods TEXT NOT NULL DEFAULT 'ff,squash',
	created_at DATETIME NOT NULL,
	UNIQUE (repo, pattern)
);

-- +goose Down
DROP TABLE branch_rules;
