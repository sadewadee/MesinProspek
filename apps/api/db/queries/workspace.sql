-- name: SetWorkspaceContext :exec
-- SetWorkspaceContext sets the row-level security context for the current
-- transaction only (is_local = true); db.InWorkspace calls it (docs/04 §4.4).
SELECT set_config('app.workspace_id', sqlc.arg(workspace_id)::text, true),
       set_config('app.user_id', sqlc.arg(user_id)::text, true);
