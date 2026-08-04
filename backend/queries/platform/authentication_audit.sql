-- name: AppendAuthenticationAuditEvent :one
-- Authentication audit evidence is append-only. No update or delete query is provided.
INSERT INTO audit_logs (
    audit_event_id,
    occurred_at,
    action,
    result,
    severity,
    actor_user_id,
    correlation_id,
    session_id,
    token_family_id,
    network_identity_hash,
    user_agent
) VALUES (
    sqlc.arg(audit_event_id),
    sqlc.arg(occurred_at),
    sqlc.arg(action),
    sqlc.arg(result),
    sqlc.arg(severity),
    sqlc.narg(actor_user_id),
    sqlc.arg(correlation_id),
    sqlc.narg(session_id),
    sqlc.narg(token_family_id),
    sqlc.narg(network_identity_hash),
    sqlc.narg(user_agent)
)
RETURNING *;
