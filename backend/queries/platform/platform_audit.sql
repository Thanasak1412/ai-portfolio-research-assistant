-- name: AppendPlatformAuditEvent :one
-- Platform audit evidence is append-only. M3 references are allowlisted UUIDs;
-- no request body, financial fields, or arbitrary metadata are accepted.
INSERT INTO audit_logs (
    audit_event_id,
    occurred_at,
    action,
    result,
    severity,
    actor_user_id,
    correlation_id,
    portfolio_id,
    transaction_id,
    correction_id
) VALUES (
    sqlc.arg(audit_event_id),
    sqlc.arg(occurred_at),
    sqlc.arg(action),
    sqlc.arg(result),
    sqlc.arg(severity),
    sqlc.narg(actor_user_id),
    sqlc.arg(correlation_id),
    sqlc.narg(portfolio_id),
    sqlc.narg(transaction_id),
    sqlc.narg(correction_id)
)
RETURNING *;
