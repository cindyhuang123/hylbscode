-- name: ListTodosBySession :many
SELECT *
FROM todos
WHERE session_id = ?
ORDER BY position ASC, created_at ASC;

-- name: GetTodoByID :one
SELECT *
FROM todos
WHERE id = ? LIMIT 1;

-- name: BulkDeleteTodosBySession :exec
DELETE FROM todos
WHERE session_id = ?;

-- name: CreateTodo :one
INSERT INTO todos (
    id,
    session_id,
    content,
    status,
    priority,
    position,
    created_at,
    updated_at,
    finished_at
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('session_id'),
    sqlc.arg('content'),
    sqlc.arg('status'),
    sqlc.arg('priority'),
    sqlc.arg('position'),
    strftime('%s', 'now'),
    strftime('%s', 'now'),
    CASE WHEN sqlc.arg('status') = 'completed' THEN strftime('%s', 'now') ELSE NULL END
) RETURNING *;

-- name: UpdateTodoStatus :one
UPDATE todos
SET
    status = sqlc.arg('status'),
    finished_at = CASE WHEN sqlc.arg('status') = 'completed' THEN strftime('%s', 'now') ELSE NULL END
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteTodo :exec
DELETE FROM todos
WHERE id = ?;
