package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/audit"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database/sqlcgen"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
)

// PlatformAuditStore maps the Platform audit abstraction to append-only sqlc
// persistence. It accepts only M3's allowlisted safe-reference record.
type PlatformAuditStore struct{ queries *sqlcgen.Queries }

func NewPlatformAuditStore(database sqlcgen.DBTX) *PlatformAuditStore {
	return &PlatformAuditStore{queries: sqlcgen.New(database)}
}

func (store *PlatformAuditStore) Append(ctx context.Context, record audit.Record) error {
	if store == nil || store.queries == nil || record.Validate() != nil {
		return audit.ErrInvalidRecord
	}
	_, err := store.queries.AppendPlatformAuditEvent(ctx, sqlcgen.AppendPlatformAuditEventParams{
		AuditEventID:  pgUUID(record.EventID),
		OccurredAt:    pgTime(record.OccurredAt),
		Action:        string(record.Action),
		Result:        string(record.Result),
		Severity:      string(record.Severity),
		ActorUserID:   pgOptionalUUID(record.ActorUserID),
		CorrelationID: record.CorrelationID,
		PortfolioID:   pgOptionalUUID(record.PortfolioID),
		TransactionID: pgOptionalUUID(record.TransactionID),
		CorrectionID:  pgOptionalUUID(record.CorrectionID),
	})
	return err
}

// PostgresOutboxStore never starts or commits a transaction. Construct it with
// a pgx.Tx to append an event atomically with an authoritative write, or with a
// pool for the later worker-facing claim operations.
type PostgresOutboxStore struct{ queries *sqlcgen.Queries }

func NewPostgresOutboxStore(database sqlcgen.DBTX) *PostgresOutboxStore {
	return &PostgresOutboxStore{queries: sqlcgen.New(database)}
}

func (store *PostgresOutboxStore) Append(ctx context.Context, event outbox.Event) error {
	if store == nil || store.queries == nil || event.Validate() != nil {
		return outbox.ErrInvalidEvent
	}
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return outbox.ErrInvalidEvent
	}
	_, err = store.queries.AppendPlatformOutboxEvent(ctx, sqlcgen.AppendPlatformOutboxEventParams{
		EventID:       pgUUID(event.ID),
		EventType:     event.Type,
		EventVersion:  event.Version,
		AggregateType: event.AggregateType,
		AggregateID:   pgUUID(event.AggregateID),
		PortfolioID:   pgUUID(event.PortfolioID),
		TransactionID: pgOptionalUUID(event.TransactionID),
		CorrectionID:  pgOptionalUUID(event.CorrectionID),
		OccurredAt:    pgTime(event.OccurredAt),
		CorrelationID: event.CorrelationID,
		Payload:       payload,
		NextAttemptAt: pgTime(event.NextAttemptAt),
	})
	return err
}

func (store *PostgresOutboxStore) ClaimDue(ctx context.Context, request outbox.ClaimRequest) ([]outbox.ClaimedEvent, error) {
	if store == nil || store.queries == nil || request.Validate() != nil {
		return nil, outbox.ErrInvalidClaimRequest
	}
	rows, err := store.queries.ClaimDuePlatformOutboxEvents(ctx, sqlcgen.ClaimDuePlatformOutboxEventsParams{
		AsOf:           pgTime(request.AsOf),
		ClaimToken:     pgUUID(request.ClaimToken),
		LeaseExpiresAt: pgTime(request.LeaseExpiresAt),
		BatchLimit:     request.BatchLimit,
	})
	if err != nil {
		return nil, err
	}
	result := make([]outbox.ClaimedEvent, 0, len(rows))
	for _, row := range rows {
		claimed, mapErr := mapClaimedOutboxEvent(row)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, claimed)
	}
	return result, nil
}

func (store *PostgresOutboxStore) MarkPublished(ctx context.Context, eventID, claimToken [16]byte, publishedAt time.Time) (bool, error) {
	if store == nil || store.queries == nil || eventID == [16]byte{} || claimToken == [16]byte{} || publishedAt.IsZero() {
		return false, outbox.ErrInvalidEvent
	}
	count, err := store.queries.MarkPlatformOutboxEventPublished(ctx, sqlcgen.MarkPlatformOutboxEventPublishedParams{
		PublishedAt: pgTime(publishedAt), EventID: pgUUID(eventID), ClaimToken: pgUUID(claimToken),
	})
	return count == 1, err
}

func (store *PostgresOutboxStore) Reschedule(ctx context.Context, eventID, claimToken [16]byte, nextAttemptAt time.Time, failureCode string) (bool, error) {
	if store == nil || store.queries == nil || eventID == [16]byte{} || claimToken == [16]byte{} || nextAttemptAt.IsZero() {
		return false, outbox.ErrInvalidEvent
	}
	if err := outbox.ValidateFailureCode(failureCode); err != nil {
		return false, err
	}
	count, err := store.queries.RescheduleClaimedPlatformOutboxEvent(ctx, sqlcgen.RescheduleClaimedPlatformOutboxEventParams{
		NextAttemptAt: pgTime(nextAttemptAt), LastFailureCode: pgText(failureCode),
		EventID: pgUUID(eventID), ClaimToken: pgUUID(claimToken),
	})
	return count == 1, err
}

func (store *PostgresOutboxStore) MarkDeadLetter(ctx context.Context, eventID, claimToken [16]byte, failureCode string) (bool, error) {
	if store == nil || store.queries == nil || eventID == [16]byte{} || claimToken == [16]byte{} {
		return false, outbox.ErrInvalidEvent
	}
	if err := outbox.ValidateFailureCode(failureCode); err != nil {
		return false, err
	}
	count, err := store.queries.MarkClaimedPlatformOutboxEventDeadLetter(ctx, sqlcgen.MarkClaimedPlatformOutboxEventDeadLetterParams{
		LastFailureCode: pgText(failureCode), EventID: pgUUID(eventID), ClaimToken: pgUUID(claimToken),
	})
	return count == 1, err
}

// PostgresConsumerDeduplicator intentionally has no transaction wrapper. A
// caller may use the same pgx.Tx for its durable side effect and this marker.
type PostgresConsumerDeduplicator struct{ queries *sqlcgen.Queries }

func NewPostgresConsumerDeduplicator(database sqlcgen.DBTX) *PostgresConsumerDeduplicator {
	return &PostgresConsumerDeduplicator{queries: sqlcgen.New(database)}
}

func (store *PostgresConsumerDeduplicator) RecordIfNew(ctx context.Context, consumerName string, eventID [16]byte, processedAt time.Time) (bool, error) {
	if store == nil || store.queries == nil || eventID == [16]byte{} || processedAt.IsZero() {
		return false, outbox.ErrInvalidEvent
	}
	if err := outbox.ValidateConsumerName(consumerName); err != nil {
		return false, err
	}
	return store.queries.RecordPlatformConsumerDeduplication(ctx, sqlcgen.RecordPlatformConsumerDeduplicationParams{
		ConsumerName: consumerName, EventID: pgUUID(eventID), ProcessedAt: pgTime(processedAt),
	})
}

func mapClaimedOutboxEvent(row sqlcgen.PlatformOutboxEvent) (outbox.ClaimedEvent, error) {
	var payload outbox.Payload
	if err := json.Unmarshal(row.Payload, &payload); err != nil || payload.SchemaVersion <= 0 ||
		!row.EventID.Valid || !row.AggregateID.Valid || !row.PortfolioID.Valid ||
		!row.OccurredAt.Valid || !row.NextAttemptAt.Valid || !row.ClaimToken.Valid ||
		!row.ClaimedAt.Valid || !row.LeaseExpiresAt.Valid {
		return outbox.ClaimedEvent{}, fmt.Errorf("map platform outbox event: %w", outbox.ErrInvalidEvent)
	}
	return outbox.ClaimedEvent{
		Event: outbox.Event{
			ID: row.EventID.Bytes, Type: row.EventType, Version: row.EventVersion,
			AggregateType: row.AggregateType, AggregateID: row.AggregateID.Bytes,
			PortfolioID: row.PortfolioID.Bytes, TransactionID: optionalUUID(row.TransactionID),
			CorrectionID: optionalUUID(row.CorrectionID), OccurredAt: row.OccurredAt.Time,
			CorrelationID: row.CorrelationID, Payload: payload, NextAttemptAt: row.NextAttemptAt.Time,
		},
		AttemptCount: row.AttemptCount, ClaimToken: row.ClaimToken.Bytes,
		ClaimedAt: row.ClaimedAt.Time, LeaseExpiresAt: row.LeaseExpiresAt.Time,
	}, nil
}

func optionalUUID(value pgtype.UUID) *[16]byte {
	if !value.Valid {
		return nil
	}
	result := value.Bytes
	return &result
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

var _ audit.Store = (*PlatformAuditStore)(nil)
var _ outbox.DeliveryStore = (*PostgresOutboxStore)(nil)
var _ outbox.ConsumerDeduplicator = (*PostgresConsumerDeduplicator)(nil)
