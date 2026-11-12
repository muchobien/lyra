-- name: CreateEvent :one
INSERT INTO events (at, action, repo, item, user, via, agent, session, data)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

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

-- name: ListEventsAfter :many
SELECT * FROM events WHERE id > ? ORDER BY id LIMIT ?;

-- name: LatestEventID :one
SELECT CAST(COALESCE(MAX(id), 0) AS INTEGER) AS id FROM events;

-- name: TokenActive :one
SELECT COUNT(*) FROM tokens WHERE id = ? AND (expires_at IS NULL OR expires_at > ?);

-- name: CreateItem :one
INSERT INTO items (repo, number, kind, title, body, state, user, via, agent, session, created_at, updated_at)
VALUES (
	sqlc.arg(repo),
	(SELECT COALESCE(MAX(number), 0) + 1 FROM items WHERE repo = sqlc.arg(repo)),
	sqlc.arg(kind), sqlc.arg(title), sqlc.arg(body), 'open',
	sqlc.arg(user), sqlc.arg(via), sqlc.arg(agent), sqlc.arg(session),
	sqlc.arg(created_at), sqlc.arg(created_at)
)
RETURNING *;

-- name: GetItem :one
SELECT * FROM items WHERE repo = ? AND number = ?;

-- name: ListItems :many
SELECT * FROM items
WHERE repo = sqlc.arg(repo) AND kind = sqlc.arg(kind)
	AND (sqlc.arg(state) = 'all' OR state = sqlc.arg(state))
ORDER BY number DESC;

-- name: UpdateItem :one
UPDATE items SET title = ?, body = ?, state = ?, updated_at = ?, closed_at = ?
WHERE id = ?
RETURNING *;

-- name: TouchItem :exec
UPDATE items SET updated_at = ? WHERE id = ?;

-- name: ListItemLabels :many
SELECT labels.name FROM item_labels
JOIN labels ON labels.id = item_labels.label_id
WHERE item_labels.item_id = ?
ORDER BY labels.name;

-- name: ListItemAssignees :many
SELECT assignee FROM item_assignees WHERE item_id = ? ORDER BY assignee;

-- name: CountComments :one
SELECT COUNT(*) FROM comments WHERE item_id = ?;

-- name: AddItemLabel :execrows
INSERT OR IGNORE INTO item_labels (item_id, label_id) VALUES (?, ?);

-- name: RemoveItemLabel :execrows
DELETE FROM item_labels WHERE item_id = ? AND label_id = ?;

-- name: AddItemAssignee :execrows
INSERT OR IGNORE INTO item_assignees (item_id, assignee) VALUES (?, ?);

-- name: RemoveItemAssignee :execrows
DELETE FROM item_assignees WHERE item_id = ? AND assignee = ?;

-- name: CreateLabel :one
INSERT INTO labels (repo, name, color, description) VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetLabel :one
SELECT * FROM labels WHERE repo = ? AND name = ?;

-- name: ListLabels :many
SELECT * FROM labels WHERE repo = ? ORDER BY name;

-- name: DeleteLabel :execrows
DELETE FROM labels WHERE repo = ? AND name = ?;

-- name: CreateComment :one
INSERT INTO comments (item_id, body, user, via, agent, session, created_at, review_id, commit_sha, path, side, line)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListComments :many
SELECT * FROM comments WHERE item_id = ? ORDER BY id;

-- name: CreatePull :exec
INSERT INTO pulls (item_id, base, head, head_sha, base_sha) VALUES (?, ?, ?, ?, ?);

-- name: GetPullItem :one
SELECT sqlc.embed(items), sqlc.embed(pulls) FROM items
JOIN pulls ON pulls.item_id = items.id
WHERE items.repo = ? AND items.number = ?;

-- name: ListPullItems :many
SELECT sqlc.embed(items), sqlc.embed(pulls) FROM items
JOIN pulls ON pulls.item_id = items.id
WHERE items.repo = sqlc.arg(repo) AND (sqlc.arg(state) = 'all' OR items.state = sqlc.arg(state))
ORDER BY items.number DESC;

-- name: UpdatePullHead :exec
UPDATE pulls SET head_sha = ?, base_sha = ? WHERE item_id = ?;

-- name: MergePull :exec
UPDATE pulls SET merge_sha = ?, base_sha = ?, merged_at = ? WHERE item_id = ?;

-- name: SetRule :one
INSERT INTO branch_rules (repo, pattern, require_signed, require_approval, merge_methods, created_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (repo, pattern) DO UPDATE SET
	require_signed = excluded.require_signed,
	require_approval = excluded.require_approval,
	merge_methods = excluded.merge_methods
RETURNING *;

-- name: ListRules :many
SELECT * FROM branch_rules WHERE repo = ? ORDER BY pattern;

-- name: DeleteRule :execrows
DELETE FROM branch_rules WHERE repo = ? AND pattern = ?;

-- name: CreateReview :one
INSERT INTO reviews (item_id, state, body, commit_sha, user, via, agent, session, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListReviews :many
SELECT * FROM reviews WHERE item_id = ? ORDER BY id;

-- name: ListItemEvents :many
SELECT * FROM events WHERE repo = ? AND item = ? ORDER BY id;

-- name: SetWaitCursor :exec
INSERT INTO wait_cursors (repo, item, user, via, agent, session, event_id)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (repo, item, user, via, agent, session) DO UPDATE SET event_id = MAX(event_id, excluded.event_id);

-- name: ListWaitCursors :many
SELECT * FROM wait_cursors WHERE repo = ? AND item = ?;
