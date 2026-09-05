-- +goose Up
-- M3 Platform-owned audit references, transactional outbox, and consumer
-- deduplication. This migration deliberately creates no Transaction ledger
-- authority or financial data.

ALTER TABLE audit_logs
    ADD COLUMN portfolio_id uuid,
    ADD COLUMN transaction_id uuid,
    ADD COLUMN correction_id uuid;

ALTER TABLE audit_logs
    DROP CONSTRAINT audit_logs_action_valid;

ALTER TABLE audit_logs
    ADD CONSTRAINT audit_logs_action_valid CHECK (action IN (
        'registration_success',
        'registration_failure',
        'login_success',
        'login_failure',
        'refresh_success',
        'refresh_failure',
        'logout',
        'token_family_revocation',
        'refresh_token_reuse',
        'disabled_account_rejection',
        'transaction_create_success',
        'transaction_create_failure',
        'transaction_idempotent_replay',
        'transaction_idempotency_conflict',
        'transaction_correction_initiated',
        'transaction_correction_completed',
        'transaction_correction_rejected',
        'transaction_reversal_created',
        'transaction_replacement_created',
        'transaction_ownership_rejection'
    ));

-- M1 actions keep their existing result semantics. M3 action/result pairs are
-- explicit evidence, preventing a successful conflict or failed completion
-- record from entering the Platform audit log.
ALTER TABLE audit_logs
    ADD CONSTRAINT audit_logs_m3_action_result_valid CHECK (
        action NOT IN (
            'transaction_create_success',
            'transaction_create_failure',
            'transaction_idempotent_replay',
            'transaction_idempotency_conflict',
            'transaction_correction_initiated',
            'transaction_correction_completed',
            'transaction_correction_rejected',
            'transaction_reversal_created',
            'transaction_replacement_created',
            'transaction_ownership_rejection'
        )
        OR (action, result) IN (
            ('transaction_create_success', 'success'),
            ('transaction_create_failure', 'failure'),
            ('transaction_idempotent_replay', 'success'),
            ('transaction_idempotency_conflict', 'failure'),
            ('transaction_correction_initiated', 'success'),
            ('transaction_correction_completed', 'success'),
            ('transaction_correction_rejected', 'failure'),
            ('transaction_reversal_created', 'success'),
            ('transaction_replacement_created', 'success'),
            ('transaction_ownership_rejection', 'failure')
        )
    );

-- The outbox stores only event-routing and stable-reference data. Its payload
-- has a version and bounded role/UUID references; future consumers re-read
-- authoritative aggregates rather than receiving financial command bodies.
-- +goose StatementBegin
CREATE FUNCTION platform_outbox_payload_is_valid(value jsonb)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    reference jsonb;
BEGIN
    IF jsonb_typeof(value) <> 'object'
        OR NOT value ? 'schemaVersion'
        OR NOT value ? 'references'
        OR jsonb_typeof(value->'schemaVersion') <> 'number'
        OR (value->>'schemaVersion') !~ '^[1-9][0-9]*$'
        OR jsonb_typeof(value->'references') <> 'array'
        OR jsonb_array_length(value->'references') > 8
        OR value - 'schemaVersion' - 'references' <> '{}'::jsonb
        OR octet_length(value::text) > 2048 THEN
        RETURN false;
    END IF;

    FOR reference IN SELECT jsonb_array_elements(value->'references') LOOP
        IF jsonb_typeof(reference) <> 'object'
            OR reference - 'role' - 'id' <> '{}'::jsonb
            OR jsonb_typeof(reference->'role') <> 'string'
            OR jsonb_typeof(reference->'id') <> 'string'
            OR (reference->>'role') !~ '^[a-z][a-z0-9_]{0,63}$'
            OR (reference->>'id') !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
END;
$$;
-- +goose StatementEnd

-- A stream row is allocated inside the caller-owned transaction. Its position
-- cannot be retroactively inserted before an already allocated event.
CREATE TABLE platform_outbox_streams (
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    next_position bigint NOT NULL,
    CONSTRAINT platform_outbox_streams_pk PRIMARY KEY (aggregate_type, aggregate_id),
    CONSTRAINT platform_outbox_streams_type_valid CHECK (
        aggregate_type ~ '^[a-z][a-z0-9_]{0,63}$'
    ),
    CONSTRAINT platform_outbox_streams_next_position_valid CHECK (next_position > 0)
);

CREATE TABLE platform_outbox_events (
    event_id uuid PRIMARY KEY,
    event_type text NOT NULL,
    event_version integer NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    portfolio_id uuid NOT NULL,
    aggregate_position bigint NOT NULL,
    occurred_at timestamptz NOT NULL,
    correlation_id text NOT NULL,
    payload jsonb NOT NULL,
    publication_state text NOT NULL DEFAULT 'PENDING',
    attempt_count integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL,
    claimed_at timestamptz,
    claim_token uuid,
    lease_expires_at timestamptz,
    published_at timestamptz,
    last_failure_code text,
    CONSTRAINT platform_outbox_events_type_valid CHECK (
        event_type ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}$'
        AND char_length(event_type) <= 128
    ),
    CONSTRAINT platform_outbox_events_version_valid CHECK (event_version > 0),
    CONSTRAINT platform_outbox_events_aggregate_type_valid CHECK (
        aggregate_type ~ '^[a-z][a-z0-9_]{0,63}$'
    ),
    CONSTRAINT platform_outbox_events_correlation_id_valid CHECK (
        char_length(correlation_id) BETWEEN 1 AND 128
        AND correlation_id ~ '^[A-Za-z0-9._-]+$'
    ),
    CONSTRAINT platform_outbox_events_payload_valid CHECK (
        platform_outbox_payload_is_valid(payload)
    ),
    CONSTRAINT platform_outbox_events_aggregate_position_valid CHECK (aggregate_position > 0),
    CONSTRAINT platform_outbox_events_aggregate_position_unique
        UNIQUE (aggregate_type, aggregate_id, aggregate_position),
    CONSTRAINT platform_outbox_events_state_valid CHECK (
        publication_state IN ('PENDING', 'PROCESSING', 'PUBLISHED', 'DEAD_LETTER')
    ),
    CONSTRAINT platform_outbox_events_attempt_count_valid CHECK (attempt_count >= 0),
    CONSTRAINT platform_outbox_events_failure_code_valid CHECK (
        last_failure_code IS NULL
        OR (
            last_failure_code ~ '^[a-z][a-z0-9_]{0,63}$'
            AND char_length(last_failure_code) <= 64
        )
    ),
    CONSTRAINT platform_outbox_events_state_consistent CHECK (
        (
            publication_state = 'PENDING'
            AND claimed_at IS NULL
            AND claim_token IS NULL
            AND lease_expires_at IS NULL
            AND published_at IS NULL
        )
        OR (
            publication_state = 'PROCESSING'
            AND claimed_at IS NOT NULL
            AND claim_token IS NOT NULL
            AND lease_expires_at IS NOT NULL
            AND lease_expires_at > claimed_at
            AND published_at IS NULL
        )
        OR (
            publication_state = 'PUBLISHED'
            AND claimed_at IS NULL
            AND claim_token IS NULL
            AND lease_expires_at IS NULL
            AND published_at IS NOT NULL
            AND published_at >= occurred_at
        )
        OR (
            publication_state = 'DEAD_LETTER'
            AND claimed_at IS NULL
            AND claim_token IS NULL
            AND lease_expires_at IS NULL
            AND published_at IS NULL
            AND last_failure_code IS NOT NULL
        )
    )
);

-- Claims preserve order within an aggregate stream while allowing unrelated
-- aggregate streams to be claimed concurrently by independently scaled workers.
CREATE INDEX platform_outbox_events_claim_pending_idx
    ON platform_outbox_events (publication_state, next_attempt_at, aggregate_type, aggregate_id, aggregate_position);
CREATE INDEX platform_outbox_events_claim_lease_idx
    ON platform_outbox_events (publication_state, lease_expires_at, aggregate_type, aggregate_id, aggregate_position);
CREATE INDEX platform_outbox_events_aggregate_order_idx
    ON platform_outbox_events (aggregate_type, aggregate_id, publication_state, aggregate_position);

-- A successful insert establishes that a named consumer owns this event's
-- durable deduplication record. Callers may pass their own pgx transaction so
-- the eventual consumer side effect and this row can commit atomically.
CREATE TABLE platform_consumer_deduplications (
    consumer_name text NOT NULL,
    event_id uuid NOT NULL,
    processed_at timestamptz NOT NULL,
    CONSTRAINT platform_consumer_deduplications_pk PRIMARY KEY (consumer_name, event_id),
    CONSTRAINT platform_consumer_deduplications_name_valid CHECK (
        consumer_name ~ '^[a-z][a-z0-9_]{0,127}$'
        AND char_length(consumer_name) BETWEEN 1 AND 128
    )
);

CREATE INDEX platform_consumer_deduplications_event_idx
    ON platform_consumer_deduplications (event_id, processed_at);

-- +goose Down
-- Production rollback must not silently discard audit or delivery evidence.
-- Empty disposable databases can still run the normal down/up verification.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM audit_logs
        WHERE action IN (
            'transaction_create_success',
            'transaction_create_failure',
            'transaction_idempotent_replay',
            'transaction_idempotency_conflict',
            'transaction_correction_initiated',
            'transaction_correction_completed',
            'transaction_correction_rejected',
            'transaction_reversal_created',
            'transaction_replacement_created',
            'transaction_ownership_rejection'
        )
    ) OR EXISTS (SELECT 1 FROM platform_outbox_events)
      OR EXISTS (SELECT 1 FROM platform_outbox_streams)
      OR EXISTS (SELECT 1 FROM platform_consumer_deduplications) THEN
        RAISE EXCEPTION 'cannot roll back M3 Platform eventing while durable audit or delivery evidence exists';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP TABLE platform_consumer_deduplications;
DROP TABLE platform_outbox_events;
DROP TABLE platform_outbox_streams;
DROP FUNCTION platform_outbox_payload_is_valid(jsonb);

ALTER TABLE audit_logs
    DROP CONSTRAINT audit_logs_m3_action_result_valid;

ALTER TABLE audit_logs
    DROP CONSTRAINT audit_logs_action_valid;

ALTER TABLE audit_logs
    DROP COLUMN correction_id,
    DROP COLUMN transaction_id,
    DROP COLUMN portfolio_id;

ALTER TABLE audit_logs
    ADD CONSTRAINT audit_logs_action_valid CHECK (action IN (
        'registration_success',
        'registration_failure',
        'login_success',
        'login_failure',
        'refresh_success',
        'refresh_failure',
        'logout',
        'token_family_revocation',
        'refresh_token_reuse',
        'disabled_account_rejection'
    ));
