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
INSERT INTO branch_rules (repo, pattern, require_signed, require_approval, require_checks, allow_force_push, allow_deletion, merge_methods, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (repo, pattern) DO UPDATE SET
	allow_force_push = excluded.allow_force_push,
	allow_deletion = excluded.allow_deletion,
	require_signed = excluded.require_signed,
	require_approval = excluded.require_approval,
	require_checks = excluded.require_checks,
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

-- name: CreateRelease :one
INSERT INTO releases (repo, tag, title, notes, draft, prerelease, target, commit_sha, user, via, agent, session, created_at, published_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetRelease :one
SELECT * FROM releases WHERE repo = ? AND id = ?;

-- name: GetReleaseByTag :one
SELECT * FROM releases WHERE repo = ? AND tag = ?;

-- name: ListReleases :many
SELECT * FROM releases WHERE repo = ?
ORDER BY published_at IS NULL DESC, published_at DESC, id DESC;

-- name: LatestRelease :one
SELECT * FROM releases WHERE repo = ? AND NOT draft AND NOT prerelease
ORDER BY published_at DESC, id DESC
LIMIT 1;

-- name: UpdateRelease :one
UPDATE releases SET title = ?, notes = ?, draft = ?, prerelease = ?, target = ?, commit_sha = ?, published_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteRelease :execrows
DELETE FROM releases WHERE id = ?;

-- name: CreateAsset :one
INSERT INTO release_assets (release_id, name, object, size, sha256, content_type, user, via, agent, session, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetAsset :one
SELECT * FROM release_assets WHERE release_id = ? AND name = ?;

-- name: ListAssets :many
SELECT * FROM release_assets WHERE release_id = ? ORDER BY name;

-- name: DeleteAsset :execrows
DELETE FROM release_assets WHERE id = ?;

-- name: SetImageTag :one
INSERT INTO images (repo, name, tag, digest, media_type, user, via, agent, session, pushed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (name, tag) DO UPDATE SET
	digest = excluded.digest, media_type = excluded.media_type, user = excluded.user, via = excluded.via,
	agent = excluded.agent, session = excluded.session, pushed_at = excluded.pushed_at
RETURNING *;

-- name: DeleteImageTag :many
DELETE FROM images WHERE name = ? AND tag = ? RETURNING tag;

-- name: DeleteImageDigest :many
DELETE FROM images WHERE name = ? AND digest = ? RETURNING tag;

-- name: ListImages :many
SELECT * FROM images WHERE repo = ? ORDER BY name, pushed_at DESC, tag;

-- name: CreateRunnerToken :one
INSERT INTO runner_tokens (name, hash, prefix, user, created_at) VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: GetRunnerTokenByHash :one
SELECT * FROM runner_tokens WHERE hash = ?;

-- name: ListRunnerTokens :many
SELECT * FROM runner_tokens ORDER BY id;

-- name: DeleteRunnerToken :execrows
DELETE FROM runner_tokens WHERE id = ?;

-- name: CreateRunner :one
INSERT INTO runners (uuid, token_hash, name, version, ephemeral, registration_token_id, user, created_at, last_seen_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetRunnerByUUID :one
SELECT * FROM runners WHERE uuid = ?;

-- name: DeclareRunner :one
UPDATE runners SET labels = ?, version = ?, last_seen_at = ? WHERE id = ? RETURNING *;

-- name: TouchRunner :exec
UPDATE runners SET last_seen_at = ? WHERE id = ?;

-- name: ListRunners :many
SELECT * FROM runners ORDER BY id;

-- name: DeleteRunner :execrows
DELETE FROM runners WHERE id = ?;

-- name: CreateOwner :one
INSERT INTO owners (name, description, user, created_at) VALUES (?, ?, ?, ?) RETURNING *;

-- name: GetOwner :one
SELECT * FROM owners WHERE name = ?;

-- name: ListOwners :many
SELECT * FROM owners ORDER BY name;

-- name: DeleteOwner :execrows
DELETE FROM owners WHERE name = ?;

-- name: CreateRun :one
INSERT INTO runs (repo, number, workflow, name, title, event, ref, sha, payload, status, user, via, agent, session, created_at, pull, head_ref, base_ref)
VALUES (
	sqlc.arg(repo),
	(SELECT COALESCE(MAX(number), 0) + 1 FROM runs WHERE repo = sqlc.arg(repo)),
	sqlc.arg(workflow), sqlc.arg(name), sqlc.arg(title), sqlc.arg(event), sqlc.arg(ref), sqlc.arg(sha), sqlc.arg(payload), 'waiting',
	sqlc.arg(user), sqlc.arg(via), sqlc.arg(agent), sqlc.arg(session), sqlc.arg(created_at),
	sqlc.arg(pull), sqlc.arg(head_ref), sqlc.arg(base_ref)
)
RETURNING *;

-- name: GetRun :one
SELECT * FROM runs WHERE repo = ? AND number = ?;

-- name: GetRunByID :one
SELECT * FROM runs WHERE id = ?;

-- name: ListRuns :many
SELECT * FROM runs WHERE repo = ? ORDER BY number DESC LIMIT ?;

-- name: UpdateRunStatus :exec
UPDATE runs SET status = ?, started_at = ?, stopped_at = ? WHERE id = ?;

-- name: CreateJob :one
INSERT INTO jobs (run_id, job_key, name, needs, runs_on, conditional, payload, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListRunJobs :many
SELECT * FROM jobs WHERE run_id = ? ORDER BY id;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = ?;

-- name: ListWaitingJobs :many
SELECT * FROM jobs WHERE status = 'waiting' ORDER BY id;

-- name: UpdateJob :exec
UPDATE jobs SET status = ?, attempt = ?, outputs = ?, started_at = ?, stopped_at = ? WHERE id = ?;

-- name: CreateTask :one
INSERT INTO tasks (job_id, runner_id, attempt, token_hash, status, started_at, updated_at)
VALUES (?, ?, ?, ?, 'running', sqlc.arg(started_at), sqlc.arg(started_at))
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ?;

-- name: GetRunningTaskByToken :one
SELECT * FROM tasks WHERE token_hash = ? AND status = 'running';

-- name: ListJobTasks :many
SELECT * FROM tasks WHERE job_id = ? ORDER BY attempt;

-- name: ListRepoRunJobs :many
SELECT jobs.* FROM jobs JOIN runs ON runs.id = jobs.run_id
WHERE runs.repo = ? AND runs.number >= ? ORDER BY jobs.id;

-- name: ListRepoRunTasks :many
SELECT tasks.* FROM tasks JOIN jobs ON jobs.id = tasks.job_id JOIN runs ON runs.id = jobs.run_id
WHERE runs.repo = ? AND runs.number >= ? ORDER BY tasks.job_id, tasks.attempt;

-- name: UpdateTask :exec
UPDATE tasks SET status = ?, steps = ?, stopped_at = ?, updated_at = ? WHERE id = ?;

-- name: SetTaskLog :exec
UPDATE tasks SET log_length = ?, log_object = ? WHERE id = ?;

-- name: InsertLogRow :exec
INSERT INTO log_rows (task_id, idx, at, content) VALUES (?, ?, ?, ?);

-- name: ListLogRows :many
SELECT * FROM log_rows WHERE task_id = ? AND idx >= ? ORDER BY idx;

-- name: DeleteLogRows :exec
DELETE FROM log_rows WHERE task_id = ?;

-- name: ListRunsForSHA :many
SELECT * FROM runs WHERE repo = ? AND sha = ? ORDER BY number;

-- name: SetSecret :exec
INSERT INTO secrets (repo, name, value, user, updated_at) VALUES (?, ?, ?, ?, ?)
ON CONFLICT (repo, name) DO UPDATE SET value = excluded.value, user = excluded.user, updated_at = excluded.updated_at;

-- name: ListSecrets :many
SELECT * FROM secrets WHERE repo = ? ORDER BY name;

-- name: DeleteSecret :execrows
DELETE FROM secrets WHERE repo = ? AND name = ?;

-- name: SetVariable :exec
INSERT INTO variables (repo, name, value, user, updated_at) VALUES (?, ?, ?, ?, ?)
ON CONFLICT (repo, name) DO UPDATE SET value = excluded.value, user = excluded.user, updated_at = excluded.updated_at;

-- name: ListVariables :many
SELECT * FROM variables WHERE repo = ? ORDER BY name;

-- name: DeleteVariable :execrows
DELETE FROM variables WHERE repo = ? AND name = ?;

-- name: ListAllAssets :many
SELECT releases.repo, releases.tag, release_assets.name, release_assets.object, release_assets.size, release_assets.sha256
FROM release_assets JOIN releases ON releases.id = release_assets.release_id
ORDER BY releases.repo, releases.tag, release_assets.name;

-- name: ListLogObjects :many
SELECT runs.repo, tasks.id, tasks.log_object
FROM tasks JOIN jobs ON jobs.id = tasks.job_id JOIN runs ON runs.id = jobs.run_id
WHERE tasks.log_object != ''
ORDER BY tasks.id;

-- name: ListAllImages :many
SELECT * FROM images ORDER BY name, tag;

-- name: LatestEventByAction :one
SELECT * FROM events WHERE action = ? ORDER BY id DESC LIMIT 1;

-- name: TouchTask :exec
UPDATE tasks SET updated_at = ? WHERE id = ?;

-- name: ListStaleTasks :many
SELECT id FROM tasks WHERE status = 'running' AND updated_at < ? ORDER BY id;

-- name: ListExpiredRuns :many
SELECT id, repo FROM runs WHERE stopped_at IS NOT NULL AND stopped_at < ? ORDER BY id;

-- name: ListRunLogObjects :many
SELECT tasks.log_object FROM tasks JOIN jobs ON jobs.id = tasks.job_id WHERE jobs.run_id = ? AND tasks.log_object != '';

-- name: DeleteRun :exec
DELETE FROM runs WHERE id = ?;

-- name: CountRunsByStatus :many
SELECT status, COUNT(*) AS count FROM runs GROUP BY status;

-- name: CountJobsByStatus :many
SELECT status, COUNT(*) AS count FROM jobs GROUP BY status;

-- name: CountRepos :one
SELECT COUNT(*) FROM repos;
