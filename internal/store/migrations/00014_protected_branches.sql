-- +goose Up
ALTER TABLE branch_rules ADD COLUMN allow_force_push BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE branch_rules ADD COLUMN allow_deletion BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE branch_rules DROP COLUMN allow_deletion;
ALTER TABLE branch_rules DROP COLUMN allow_force_push;
