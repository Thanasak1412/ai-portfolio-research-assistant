-- name: RecordPlatformConsumerDeduplication :one
-- The caller controls the DBTX so this durable marker can commit atomically
-- with the consuming side effect. true means this consumer has not processed
-- this event before; false means an existing marker won the race.
WITH inserted AS (
    INSERT INTO platform_consumer_deduplications (
        consumer_name,
        event_id,
        processed_at
    ) VALUES (
        sqlc.arg(consumer_name),
        sqlc.arg(event_id),
        sqlc.arg(processed_at)
    )
    ON CONFLICT (consumer_name, event_id) DO NOTHING
    RETURNING 1
)
SELECT EXISTS (SELECT 1 FROM inserted) AS inserted;
