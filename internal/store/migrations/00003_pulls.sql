-- +goose Up
CREATE TABLE pulls (
	item_id INTEGER PRIMARY KEY REFERENCES items (id) ON DELETE CASCADE,
	base TEXT NOT NULL,
	head TEXT NOT NULL,
	head_sha TEXT NOT NULL,
	base_sha TEXT NOT NULL,
	merge_sha TEXT NOT NULL DEFAULT '',
	merged_at DATETIME
);

-- +goose Down
DROP TABLE pulls;
