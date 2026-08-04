-- name: AcquireAuthRateLimitAdvisoryLock :exec
-- Must run inside the rate-limit transaction. lock_key is deterministically derived by the future
-- runtime adapter from the policy namespace and HMAC-derived key; raw email/IP input is forbidden.
SELECT pg_advisory_xact_lock(sqlc.arg(lock_key)::bigint);

-- name: DeleteExpiredAuthRateLimitEventsForKey :execrows
DELETE FROM auth_rate_limit_events
WHERE
    policy_name = sqlc.arg(policy_name)
    AND policy_version = sqlc.arg(policy_version)
    AND derived_key = sqlc.arg(derived_key)
    AND expires_at <= sqlc.arg(as_of);

-- name: CountActiveAuthRateLimitEvents :one
SELECT count(*)
FROM auth_rate_limit_events
WHERE
    policy_name = sqlc.arg(policy_name)
    AND policy_version = sqlc.arg(policy_version)
    AND derived_key = sqlc.arg(derived_key)
    AND expires_at > sqlc.arg(as_of);

-- name: InsertAuthRateLimitEvent :one
INSERT INTO auth_rate_limit_events (
    derived_key,
    policy_name,
    policy_version,
    occurred_at,
    expires_at
) VALUES (
    sqlc.arg(derived_key),
    sqlc.arg(policy_name),
    sqlc.arg(policy_version),
    sqlc.arg(occurred_at),
    sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetEarliestActiveAuthRateLimitExpiry :one
SELECT min(expires_at)::timestamptz
FROM auth_rate_limit_events
WHERE
    policy_name = sqlc.arg(policy_name)
    AND policy_version = sqlc.arg(policy_version)
    AND derived_key = sqlc.arg(derived_key)
    AND expires_at > sqlc.arg(as_of);

-- name: ClearLoginEmailFailureEvents :execrows
DELETE FROM auth_rate_limit_events
WHERE
    policy_name = 'login_email_failure'
    AND policy_version = sqlc.arg(policy_version)
    AND derived_key = sqlc.arg(derived_key);

-- name: DeleteGloballyExpiredAuthRateLimitEvents :execrows
DELETE FROM auth_rate_limit_events
WHERE expires_at <= sqlc.arg(as_of);
