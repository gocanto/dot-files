-- name: CreateRun :exec
INSERT INTO workflow_runs (
  id,
  workflow_id,
  workflow_name,
  confirmation_option_id,
  confirmation_option_label,
  mode,
  status,
  started_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: CompleteRun :exec
UPDATE workflow_runs
SET status = ?, completed_at = ?, error_message = ?
WHERE id = ?;

-- name: InsertEvent :exec
INSERT INTO workflow_events (
  run_id,
  seq,
  event_type,
  phase_id,
  phase_name,
  status,
  message,
  created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListRuns :many
SELECT
  id,
  workflow_id,
  workflow_name,
  confirmation_option_id,
  confirmation_option_label,
  mode,
  status,
  started_at,
  completed_at,
  error_message
FROM workflow_runs
ORDER BY started_at DESC
LIMIT ?;

-- name: GetRun :one
SELECT
  id,
  workflow_id,
  workflow_name,
  confirmation_option_id,
  confirmation_option_label,
  mode,
  status,
  started_at,
  completed_at,
  error_message
FROM workflow_runs
WHERE id = ?;

-- name: ListRunEvents :many
SELECT
  id,
  run_id,
  seq,
  event_type,
  phase_id,
  phase_name,
  status,
  message,
  created_at
FROM workflow_events
WHERE run_id = ?
ORDER BY seq ASC;

-- name: GetUserPreferences :one
SELECT theme, updated_at
FROM user_preferences
WHERE id = 1;

-- name: UpsertUserPreferences :exec
INSERT INTO user_preferences (id, theme, updated_at)
VALUES (1, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  theme = excluded.theme,
  updated_at = excluded.updated_at;
