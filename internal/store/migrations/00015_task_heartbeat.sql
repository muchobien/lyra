-- +goose Up
ALTER TABLE tasks ADD COLUMN updated_at DATETIME;
UPDATE tasks SET updated_at = started_at;
CREATE INDEX tasks_running ON tasks (status, updated_at);

-- +goose Down
DROP INDEX tasks_running;
ALTER TABLE tasks DROP COLUMN updated_at;
