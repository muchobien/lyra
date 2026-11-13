-- +goose Up
ALTER TABLE runs ADD COLUMN pull INTEGER NOT NULL DEFAULT 0;
ALTER TABLE runs ADD COLUMN head_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE runs ADD COLUMN base_ref TEXT NOT NULL DEFAULT '';
CREATE INDEX runs_sha ON runs (repo, sha);
ALTER TABLE branch_rules ADD COLUMN require_checks BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE branch_rules DROP COLUMN require_checks;
DROP INDEX runs_sha;
ALTER TABLE runs DROP COLUMN base_ref;
ALTER TABLE runs DROP COLUMN head_ref;
ALTER TABLE runs DROP COLUMN pull;
