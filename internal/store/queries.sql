-- name: CreateEvent :exec
INSERT INTO events (at, action, repo, user, via, agent, session, data)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListEvents :many
SELECT * FROM events ORDER BY id DESC LIMIT ?;

-- name: ListRepoEvents :many
SELECT * FROM events WHERE repo = ? ORDER BY id DESC LIMIT ?;

-- name: CreateRepo :one
INSERT INTO repos (name, description, created_at, created_by)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetRepo :one
SELECT * FROM repos WHERE name = ?;

-- name: ListRepos :many
SELECT * FROM repos ORDER BY name;

-- name: DeleteRepo :execrows
DELETE FROM repos WHERE name = ?;

-- name: CreateToken :one
INSERT INTO tokens (name, user, hash, prefix, scope, repos, created_at, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UseToken :one
UPDATE tokens SET last_used_at = ? WHERE hash = ? RETURNING *;

-- name: ListTokens :many
SELECT * FROM tokens WHERE user = ? ORDER BY id;

-- name: DeleteToken :execrows
DELETE FROM tokens WHERE user = ? AND id = ?;

-- name: CreateKey :one
INSERT INTO keys (user, title, agent, public_key, fingerprint, signing, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListKeys :many
SELECT * FROM keys ORDER BY id;

-- name: ListUserKeys :many
SELECT * FROM keys WHERE user = ? ORDER BY id;

-- name: DeleteKey :execrows
DELETE FROM keys WHERE user = ? AND id = ?;
