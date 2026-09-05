-- name: AppendPlatformOutboxEvent :one
-- The caller controls the DBTX. Supplying a pgx.Tx makes this append part of
-- the caller's authoritative write transaction.
INSERT INTO platform_outbox_events (
    event_id,
    event_type,
    event_version,
    aggregate_type,
    aggregate_id,
    portfolio_id,
    transaction_id,
    correction_id,
    occurred_at,
    correlation_id,
    payload,
    publication_state,
    attempt_count,
    next_attempt_at
) VALUES (
    sqlc.arg(event_id),
    sqlc.arg(event_type),
    sqlc.arg(event_version),
    sqlc.arg(aggregate_type),
    sqlc.arg(aggregate_id),
    sqlc.arg(portfolio_id),
    sqlc.narg(transaction_id),
    sqlc.narg(correction_id),
    sqlc.arg(occurred_at),
    sqlc.arg(correlation_id),
    sqlc.arg(payload),
    'PENDING',
    0,
    sqlc.arg(next_attempt_at)
)
RETURNING *;

-- name: ClaimDuePlatformOutboxEvents :many
-- This single statement uses SKIP LOCKED and an aggregate-stream predecessor
-- check, so concurrent workers cannot own the same live claim or overtake an
-- unpublished predecessor in one aggregate stream.
WITH candidates AS (
    SELECT candidate.event_id
    FROM platform_outbox_events AS candidate
    WHERE (
        (candidate.publication_state = 'PENDING' AND candidate.next_attempt_at <= sqlc.arg(as_of))
        OR (
            candidate.publication_state = 'PROCESSING'
            AND candidate.lease_expires_at <= sqlc.arg(as_of)
        )
    )
    AND NOT EXISTS (
        SELECT 1
        FROM platform_outbox_events AS predecessor
        WHERE predecessor.aggregate_type = candidate.aggregate_type
          AND predecessor.aggregate_id = candidate.aggregate_id
          AND predecessor.publication_state <> 'PUBLISHED'
          AND (predecessor.occurred_at, predecessor.event_id)
              < (candidate.occurred_at, candidate.event_id)
    )
    ORDER BY candidate.occurred_at ASC, candidate.event_id ASC
    LIMIT sqlc.arg(batch_limit)::integer
    FOR UPDATE SKIP LOCKED
)
UPDATE platform_outbox_events AS event
SET publication_state = 'PROCESSING',
    attempt_count = event.attempt_count + 1,
    claimed_at = sqlc.arg(as_of),
    claim_token = sqlc.arg(claim_token),
    lease_expires_at = sqlc.arg(lease_expires_at)
FROM candidates
WHERE event.event_id = candidates.event_id
RETURNING event.*;

-- name: MarkPlatformOutboxEventPublished :execrows
UPDATE platform_outbox_events
SET publication_state = 'PUBLISHED',
    claimed_at = NULL,
    claim_token = NULL,
    lease_expires_at = NULL,
    published_at = sqlc.arg(published_at),
    last_failure_code = NULL
WHERE event_id = sqlc.arg(event_id)
  AND publication_state = 'PROCESSING'
  AND claim_token = sqlc.arg(claim_token);

-- name: RescheduleClaimedPlatformOutboxEvent :execrows
UPDATE platform_outbox_events
SET publication_state = 'PENDING',
    next_attempt_at = sqlc.arg(next_attempt_at),
    claimed_at = NULL,
    claim_token = NULL,
    lease_expires_at = NULL,
    last_failure_code = sqlc.arg(last_failure_code)
WHERE event_id = sqlc.arg(event_id)
  AND publication_state = 'PROCESSING'
  AND claim_token = sqlc.arg(claim_token);

-- name: MarkClaimedPlatformOutboxEventDeadLetter :execrows
UPDATE platform_outbox_events
SET publication_state = 'DEAD_LETTER',
    claimed_at = NULL,
    claim_token = NULL,
    lease_expires_at = NULL,
    last_failure_code = sqlc.arg(last_failure_code)
WHERE event_id = sqlc.arg(event_id)
  AND publication_state = 'PROCESSING'
  AND claim_token = sqlc.arg(claim_token);
