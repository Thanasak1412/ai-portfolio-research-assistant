-- name: InsertRefreshSessionGeneration :one
-- Initial and replacement generations are inserted as active. Replacement insertion must run
-- in the same transaction as MarkActiveRefreshSessionReplaced.
INSERT INTO refresh_sessions (
    session_id,
    token_family_id,
    user_id,
    token_digest,
    session_state,
    replacement_session_id,
    created_at,
    replaced_at,
    idle_expires_at,
    absolute_expires_at,
    revoked_at,
    revocation_reason,
    network_identity_hash,
    user_agent
) VALUES (
    sqlc.arg(session_id),
    sqlc.arg(token_family_id),
    sqlc.arg(user_id),
    sqlc.arg(token_digest),
    'active',
    NULL,
    sqlc.arg(created_at),
    NULL,
    sqlc.arg(idle_expires_at),
    sqlc.arg(absolute_expires_at),
    NULL,
    NULL,
    sqlc.narg(network_identity_hash),
    sqlc.narg(user_agent)
)
RETURNING *;

-- name: GetRefreshSessionByTokenDigestForUpdate :one
-- Must run inside the refresh transaction before any lifecycle transition.
SELECT *
FROM refresh_sessions
WHERE token_digest = sqlc.arg(token_digest)
FOR UPDATE;

-- name: GetRefreshSessionByID :one
SELECT *
FROM refresh_sessions
WHERE session_id = sqlc.arg(session_id);

-- name: MarkActiveRefreshSessionReplaced :one
-- Must run inside the same transaction as the replacement insert. The deferred self-FK permits
-- the replacement ID to be recorded before the replacement row is inserted.
UPDATE refresh_sessions
SET
    session_state = 'replaced',
    replacement_session_id = sqlc.arg(replacement_session_id),
    replaced_at = sqlc.arg(replaced_at)
WHERE
    session_id = sqlc.arg(session_id)
    AND session_state = 'active'
RETURNING *;

-- name: RevokeRefreshTokenFamily :many
-- Must run inside the refresh/reuse transaction and affects only the supplied family.
UPDATE refresh_sessions
SET
    session_state = 'revoked',
    revoked_at = sqlc.arg(revoked_at),
    revocation_reason = sqlc.arg(revocation_reason)
WHERE
    token_family_id = sqlc.arg(token_family_id)
    AND session_state <> 'revoked'
RETURNING *;

-- name: ListRefreshTokenFamilyState :many
SELECT *
FROM refresh_sessions
WHERE token_family_id = sqlc.arg(token_family_id)
ORDER BY created_at, session_id;

-- name: MarkExpiredRefreshSessions :many
UPDATE refresh_sessions
SET session_state = 'expired'
WHERE
    session_state = 'active'
    AND (
        idle_expires_at <= sqlc.arg(as_of)
        OR absolute_expires_at <= sqlc.arg(as_of)
    )
RETURNING session_id;

-- name: DeleteInactiveRefreshSessionsBefore :execrows
-- The caller supplies a separately approved retention cutoff; this query defines no retention period.
DELETE FROM refresh_sessions
WHERE
    (session_state = 'replaced' AND replaced_at < sqlc.arg(retention_cutoff))
    OR (session_state = 'revoked' AND revoked_at < sqlc.arg(retention_cutoff))
    OR (session_state = 'expired' AND absolute_expires_at < sqlc.arg(retention_cutoff));
