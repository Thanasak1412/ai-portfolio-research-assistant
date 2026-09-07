-- name: LockTransactionIdempotency :exec
-- Caller-owned pgx.Tx, acquired BEFORE the Portfolio sequence lock.
-- PostgreSQL hashtextextended of the namespaced, length-framed text yields a
-- deterministic signed bigint lock key. Hash collisions only over-serialize;
-- the composite primary key is the final command-identity authority. Inputs
-- are canonical UUID, allowlisted scope, validated opaque key, never credentials.
SELECT pg_advisory_xact_lock(hashtextextended(
    'transaction_idempotency:v1:' || sqlc.arg(portfolio_id)::uuid::text || ':' ||
    length(sqlc.arg(command_scope)::text)::text || ':' || sqlc.arg(command_scope)::text || ':' ||
    length(sqlc.arg(idempotency_key)::text)::text || ':' || sqlc.arg(idempotency_key)::text, 0));

-- name: GetUnexpiredTransactionIdempotency :one
-- Under the advisory lock. Fingerprint comparison/replay decisions are not SQL.
SELECT * FROM transaction_idempotency
WHERE portfolio_id = sqlc.arg(portfolio_id) AND command_scope = sqlc.arg(command_scope)
    AND idempotency_key = sqlc.arg(idempotency_key) AND expires_at > sqlc.arg(as_of);

-- name: DeleteExpiredTransactionIdempotencyKey :execrows
-- Under the same advisory lock, before treating an expired key as a new command.
DELETE FROM transaction_idempotency
WHERE portfolio_id = sqlc.arg(portfolio_id) AND command_scope = sqlc.arg(command_scope)
    AND idempotency_key = sqlc.arg(idempotency_key) AND expires_at <= sqlc.arg(as_of);

-- name: InsertCompletedTransactionIdempotency :one
-- Same pgx.Tx as facts/correction/audit/outbox. completed_at is supplied at the
-- final successful-write boundary, immediately before caller commit. 365 days
-- means 8760 hours, independent of the connection's timezone/DST.
INSERT INTO transaction_idempotency (
    portfolio_id, command_scope, idempotency_key, fingerprint_digest,
    target_transaction_id, primary_transaction_id, correction_id, created_at, expires_at
) VALUES (
    sqlc.arg(portfolio_id), sqlc.arg(command_scope), sqlc.arg(idempotency_key), sqlc.arg(fingerprint_digest),
    sqlc.narg(target_transaction_id), sqlc.arg(primary_transaction_id), sqlc.narg(correction_id),
    sqlc.arg(completed_at)::timestamptz, sqlc.arg(completed_at)::timestamptz + interval '8760 hours'
) RETURNING *;

-- name: DeleteExpiredTransactionIdempotencyBatch :execrows
-- Future worker cleanup only. No ledger fact deletion. Multiple cleanup workers
-- skip each other's row locks; concurrent renewed keys are not removed.
WITH expired AS (
    SELECT d.portfolio_id, d.command_scope, d.idempotency_key FROM transaction_idempotency AS d
    WHERE d.expires_at <= sqlc.arg(as_of)
    ORDER BY d.expires_at, d.portfolio_id, d.command_scope, d.idempotency_key
    LIMIT sqlc.arg(batch_size)::int
    FOR UPDATE SKIP LOCKED
)
DELETE FROM transaction_idempotency AS i USING expired AS e
WHERE i.portfolio_id = e.portfolio_id AND i.command_scope = e.command_scope AND i.idempotency_key = e.idempotency_key;
