-- +goose Up
ALTER TABLE pulls ADD COLUMN conflicted BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE pulls DROP COLUMN conflicted;
