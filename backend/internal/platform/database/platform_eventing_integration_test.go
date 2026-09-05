//go:build integration

package database

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/audit"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database/sqlcgen"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
)

func TestM3PlatformAuditExtendsAuthenticationAuditWithoutMetadata(t *testing.T) {
	pool := openAuthenticationTestPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	portfolioID := platformEventUUID()
	transactionID := platformEventUUID()
	correctionID := platformEventUUID()

	store := NewPlatformAuditStore(pool)
	for _, action := range []audit.Action{
		audit.ActionTransactionCreateSuccess,
		audit.ActionTransactionCreateFailure,
		audit.ActionTransactionIdempotentReplay,
		audit.ActionTransactionIdempotencyConflict,
		audit.ActionTransactionCorrectionInitiated,
		audit.ActionTransactionCorrectionCompleted,
		audit.ActionTransactionCorrectionRejected,
		audit.ActionTransactionReversalCreated,
		audit.ActionTransactionReplacementCreated,
		audit.ActionTransactionOwnershipRejection,
	} {
		if err := store.Append(ctx, audit.Record{
			EventID: platformEventUUID(), OccurredAt: now,
			Action: action, Result: audit.ResultSuccess,
			Severity: audit.SeverityInfo, CorrelationID: "corr-m3-audit-" + uuid.NewString(),
			PortfolioID: &portfolioID, TransactionID: &transactionID, CorrectionID: &correctionID,
		}); err != nil {
			t.Fatalf("append M3 audit action %q: %v", action, err)
		}
	}

	// Existing Authentication action and generated append surface remain valid.
	if _, err := sqlcgen.New(pool).AppendAuthenticationAuditEvent(ctx, sqlcgen.AppendAuthenticationAuditEventParams{
		AuditEventID: platformUUID(), OccurredAt: platformTime(now), Action: "login_failure",
		Result: "failure", Severity: "warning", CorrelationID: "corr-m1-audit-" + uuid.NewString(),
	}); err != nil {
		t.Fatalf("append existing Authentication audit event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_logs (audit_event_id, occurred_at, action, result, severity, correlation_id)
		VALUES ($1, $2, 'free_form_action', 'failure', 'warning', $3)`,
		platformUUID(), now, "corr-invalid-"+uuid.NewString()); err == nil {
		t.Fatal("expected free-form audit action rejection")
	}
	assertPlatformColumnsAbsent(t, pool, "audit_logs",
		"request_body", "quantity", "unit_price", "amount", "fee", "note", "external_reference",
		"metadata", "payload", "authorization_header", "access_token", "refresh_token", "credential")
	querySurface := reflect.TypeOf(sqlcgen.New(pool))
	for _, forbidden := range []string{"UpdatePlatformAuditEvent", "DeletePlatformAuditEvent"} {
		if _, exists := querySurface.MethodByName(forbidden); exists {
			t.Fatalf("append-only Platform audit query surface exposes %s", forbidden)
		}
	}
}

func TestPlatformOutboxAppendCanJoinCallerTransaction(t *testing.T) {
	pool := openAuthenticationTestPool(t)
	clearPlatformOutboxEvents(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	event := platformOutboxEvent(now, platformEventUUID(), platformEventUUID())

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	if err := NewPostgresOutboxStore(tx).Append(ctx, event); err != nil {
		t.Fatalf("append outbox event in transaction: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback event transaction: %v", err)
	}
	if countPlatformOutboxEvent(t, pool, event.ID) != 0 {
		t.Fatal("rolled-back outbox event persisted")
	}

	store := NewPostgresOutboxStore(pool)
	if err := store.Append(ctx, event); err != nil {
		t.Fatalf("append outbox event: %v", err)
	}
	if countPlatformOutboxEvent(t, pool, event.ID) != 1 {
		t.Fatal("appended outbox event missing")
	}

	invalidVersion := event
	invalidVersion.ID = platformEventUUID()
	invalidVersion.Version = 0
	if err := store.Append(ctx, invalidVersion); err == nil {
		t.Fatal("invalid outbox event version accepted")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE platform_outbox_events
		SET publication_state = 'PROCESSING'
		WHERE event_id = $1`, pgUUID(event.ID)); err == nil {
		t.Fatal("invalid outbox state/timestamp combination accepted")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE platform_outbox_events
		SET last_failure_code = 'unsafe failure text'
		WHERE event_id = $1`, pgUUID(event.ID)); err == nil {
		t.Fatal("unsafe outbox failure metadata accepted")
	}
	assertPlatformColumnsAbsent(t, pool, "platform_outbox_events",
		"request_body", "quantity", "unit_price", "amount", "fee", "note", "external_reference",
		"authorization_header", "access_token", "refresh_token", "credential", "provider_payload")
}

func TestPlatformOutboxClaimsAreConcurrentSafeOrderedAndLeaseRecoverable(t *testing.T) {
	pool := openAuthenticationTestPool(t)
	secondPool := openAuthenticationTestPool(t)
	clearPlatformOutboxEvents(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	store := NewPostgresOutboxStore(pool)
	secondStore := NewPostgresOutboxStore(secondPool)

	// Two concurrent claimers can never own the same live event.
	concurrentEvent := platformOutboxEvent(now, platformEventUUID(), platformEventUUID())
	if err := store.Append(ctx, concurrentEvent); err != nil {
		t.Fatalf("append concurrent event: %v", err)
	}
	type claimResult struct {
		items []outbox.ClaimedEvent
		err   error
	}
	start := make(chan struct{})
	results := make(chan claimResult, 2)
	requests := []outbox.ClaimRequest{
		{AsOf: now, ClaimToken: platformEventUUID(), LeaseExpiresAt: now.Add(time.Minute), BatchLimit: 1},
		{AsOf: now, ClaimToken: platformEventUUID(), LeaseExpiresAt: now.Add(time.Minute), BatchLimit: 1},
	}
	var wait sync.WaitGroup
	for index, claimant := range []*PostgresOutboxStore{store, secondStore} {
		wait.Add(1)
		go func(store *PostgresOutboxStore, request outbox.ClaimRequest) {
			defer wait.Done()
			<-start
			items, err := store.ClaimDue(ctx, request)
			results <- claimResult{items: items, err: err}
		}(claimant, requests[index])
	}
	close(start)
	wait.Wait()
	close(results)
	claimed := make([]outbox.ClaimedEvent, 0, 1)
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent claim: %v", result.err)
		}
		claimed = append(claimed, result.items...)
	}
	if len(claimed) != 1 || claimed[0].ID != concurrentEvent.ID || claimed[0].AttemptCount != 1 {
		t.Fatalf("concurrent claims=%+v", claimed)
	}
	if ok, err := store.MarkPublished(ctx, concurrentEvent.ID, claimed[0].ClaimToken, now.Add(time.Second)); err != nil || !ok {
		t.Fatalf("mark published: ok=%v err=%v", ok, err)
	}
	items, err := store.ClaimDue(ctx, outbox.ClaimRequest{
		AsOf: now.Add(2 * time.Hour), ClaimToken: platformEventUUID(), LeaseExpiresAt: now.Add(3 * time.Hour), BatchLimit: 1,
	})
	if err != nil || len(items) != 0 {
		t.Fatalf("published event reclaimed: items=%d err=%v", len(items), err)
	}

	// A later aggregate event cannot overtake its unpublished predecessor.
	aggregateID := platformEventUUID()
	first := platformOutboxEvent(now.Add(2*time.Hour), aggregateID, platformEventUUID())
	second := platformOutboxEvent(now.Add(2*time.Hour+time.Microsecond), aggregateID, platformEventUUID())
	if err := store.Append(ctx, first); err != nil {
		t.Fatalf("append first ordered event: %v", err)
	}
	if err := store.Append(ctx, second); err != nil {
		t.Fatalf("append second ordered event: %v", err)
	}
	orderedRequest := outbox.ClaimRequest{AsOf: now.Add(2 * time.Hour), ClaimToken: platformEventUUID(), LeaseExpiresAt: now.Add(2*time.Hour + time.Minute), BatchLimit: 2}
	items, err = store.ClaimDue(ctx, orderedRequest)
	if err != nil || len(items) != 1 || items[0].ID != first.ID {
		t.Fatalf("ordered first claim: items=%+v err=%v", items, err)
	}
	if ok, err := store.MarkPublished(ctx, first.ID, items[0].ClaimToken, now.Add(2*time.Hour+time.Second)); err != nil || !ok {
		t.Fatalf("publish ordered predecessor: ok=%v err=%v", ok, err)
	}
	items, err = store.ClaimDue(ctx, outbox.ClaimRequest{AsOf: now.Add(2*time.Hour + 2*time.Microsecond), ClaimToken: platformEventUUID(), LeaseExpiresAt: now.Add(2*time.Hour + time.Minute), BatchLimit: 2})
	if err != nil || len(items) != 1 || items[0].ID != second.ID {
		t.Fatalf("ordered successor claim: items=%+v err=%v", items, err)
	}
	if ok, err := store.MarkPublished(ctx, second.ID, items[0].ClaimToken, now.Add(2*time.Hour+3*time.Microsecond)); err != nil || !ok {
		t.Fatalf("publish ordered successor: ok=%v err=%v", ok, err)
	}

	// A crashed worker's expired lease becomes reclaimable and increments attempts.
	expiring := platformOutboxEvent(now.Add(4*time.Hour), platformEventUUID(), platformEventUUID())
	if err := store.Append(ctx, expiring); err != nil {
		t.Fatalf("append expiring event: %v", err)
	}
	items, err = store.ClaimDue(ctx, outbox.ClaimRequest{AsOf: expiring.NextAttemptAt, ClaimToken: platformEventUUID(), LeaseExpiresAt: expiring.NextAttemptAt.Add(time.Minute), BatchLimit: 1})
	if err != nil || len(items) != 1 || items[0].AttemptCount != 1 {
		t.Fatalf("initial expiring claim: items=%+v err=%v", items, err)
	}
	items, err = secondStore.ClaimDue(ctx, outbox.ClaimRequest{AsOf: expiring.NextAttemptAt.Add(2 * time.Minute), ClaimToken: platformEventUUID(), LeaseExpiresAt: expiring.NextAttemptAt.Add(3 * time.Minute), BatchLimit: 1})
	if err != nil || len(items) != 1 || items[0].ID != expiring.ID || items[0].AttemptCount != 2 {
		t.Fatalf("expired lease reclaim: items=%+v err=%v", items, err)
	}
	if ok, err := secondStore.Reschedule(ctx, expiring.ID, items[0].ClaimToken, expiring.NextAttemptAt.Add(4*time.Minute), "transient_delivery_failure"); err != nil || !ok {
		t.Fatalf("reschedule claimed event: ok=%v err=%v", ok, err)
	}
	items, err = store.ClaimDue(ctx, outbox.ClaimRequest{AsOf: expiring.NextAttemptAt.Add(4 * time.Minute), ClaimToken: platformEventUUID(), LeaseExpiresAt: expiring.NextAttemptAt.Add(5 * time.Minute), BatchLimit: 1})
	if err != nil || len(items) != 1 || items[0].AttemptCount != 3 {
		t.Fatalf("rescheduled event claim: items=%+v err=%v", items, err)
	}
	if ok, err := store.MarkDeadLetter(ctx, expiring.ID, items[0].ClaimToken, "manual_review_required"); err != nil || !ok {
		t.Fatalf("mark dead letter: ok=%v err=%v", ok, err)
	}
	items, err = store.ClaimDue(ctx, outbox.ClaimRequest{AsOf: expiring.NextAttemptAt.Add(6 * time.Minute), ClaimToken: platformEventUUID(), LeaseExpiresAt: expiring.NextAttemptAt.Add(7 * time.Minute), BatchLimit: 1})
	if err != nil || len(items) != 0 {
		t.Fatalf("dead-letter event reclaimed: items=%d err=%v", len(items), err)
	}
}

func TestM3PlatformEventingSchemaOwnsOnlyPlatformObjects(t *testing.T) {
	pool := openAuthenticationTestPool(t)
	ctx := context.Background()

	for _, relation := range []string{
		"platform_outbox_events",
		"platform_consumer_deduplications",
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, relation).Scan(&exists); err != nil {
			t.Fatalf("read relation %s: %v", relation, err)
		}
		if !exists {
			t.Fatalf("required Platform relation %s is missing", relation)
		}
	}
	for _, index := range []string{
		"platform_outbox_events_claim_pending_idx",
		"platform_outbox_events_claim_lease_idx",
		"platform_outbox_events_aggregate_order_idx",
		"platform_consumer_deduplications_event_idx",
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, index).Scan(&exists); err != nil {
			t.Fatalf("read index %s: %v", index, err)
		}
		if !exists {
			t.Fatalf("required Platform index %s is missing", index)
		}
	}
	for _, forbidden := range []string{"transactions", "transaction_idempotency"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, forbidden).Scan(&exists); err != nil {
			t.Fatalf("read relation %s: %v", forbidden, err)
		}
		if exists {
			t.Fatalf("M3-PLATFORM-001 must not create %s", forbidden)
		}
	}
}

func TestPlatformConsumerDedupIsAtomicAndConcurrent(t *testing.T) {
	pool := openAuthenticationTestPool(t)
	secondPool := openAuthenticationTestPool(t)
	clearPlatformConsumerDeduplications(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	store := NewPostgresConsumerDeduplicator(pool)
	eventID := platformEventUUID()

	if inserted, err := store.RecordIfNew(ctx, "holding_projection_v1", eventID, now); err != nil || !inserted {
		t.Fatalf("first dedup record: inserted=%v err=%v", inserted, err)
	}
	if inserted, err := store.RecordIfNew(ctx, "holding_projection_v1", eventID, now); err != nil || inserted {
		t.Fatalf("duplicate dedup record: inserted=%v err=%v", inserted, err)
	}
	if inserted, err := store.RecordIfNew(ctx, "valuation_projection_v1", eventID, now); err != nil || !inserted {
		t.Fatalf("different consumer dedup record: inserted=%v err=%v", inserted, err)
	}
	if inserted, err := store.RecordIfNew(ctx, "holding_projection_v1", platformEventUUID(), now); err != nil || !inserted {
		t.Fatalf("different event dedup record: inserted=%v err=%v", inserted, err)
	}

	// A caller-owned transaction can atomically decide whether to apply a side
	// effect. Rolling it back leaves no durable dedup marker.
	rollbackEventID := platformEventUUID()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatalf("begin dedup transaction: %v", err)
	}
	if inserted, err := NewPostgresConsumerDeduplicator(tx).RecordIfNew(ctx, "rollback_consumer_v1", rollbackEventID, now); err != nil || !inserted {
		t.Fatalf("transactional dedup insert: inserted=%v err=%v", inserted, err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback dedup transaction: %v", err)
	}
	if inserted, err := store.RecordIfNew(ctx, "rollback_consumer_v1", rollbackEventID, now); err != nil || !inserted {
		t.Fatalf("dedup marker survived rollback: inserted=%v err=%v", inserted, err)
	}

	concurrentEventID := platformEventUUID()
	start := make(chan struct{})
	results := make(chan bool, 2)
	errors := make(chan error, 2)
	var wait sync.WaitGroup
	for _, deduplicator := range []*PostgresConsumerDeduplicator{store, NewPostgresConsumerDeduplicator(secondPool)} {
		wait.Add(1)
		go func(deduplicator *PostgresConsumerDeduplicator) {
			defer wait.Done()
			<-start
			inserted, err := deduplicator.RecordIfNew(ctx, "concurrent_consumer_v1", concurrentEventID, now)
			if err != nil {
				errors <- err
				return
			}
			results <- inserted
		}(deduplicator)
	}
	close(start)
	wait.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Fatalf("concurrent dedup: %v", err)
	}
	insertedCount := 0
	for inserted := range results {
		if inserted {
			insertedCount++
		}
	}
	if insertedCount != 1 {
		t.Fatalf("concurrent dedup inserts=%d, want 1", insertedCount)
	}
}

func platformOutboxEvent(now time.Time, aggregateID, eventID [16]byte) outbox.Event {
	return outbox.Event{
		ID: eventID, Type: "transaction.recorded.v1", Version: 1,
		AggregateType: "portfolio", AggregateID: aggregateID, PortfolioID: platformEventUUID(),
		OccurredAt: now, CorrelationID: "corr-outbox-" + uuid.NewString(),
		Payload: outbox.Payload{SchemaVersion: 1}, NextAttemptAt: now,
	}
}

func platformEventUUID() [16]byte {
	value := uuid.New()
	var result [16]byte
	copy(result[:], value[:])
	return result
}

func countPlatformOutboxEvent(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, eventID [16]byte) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM platform_outbox_events WHERE event_id = $1`, pgUUID(eventID)).Scan(&count); err != nil {
		t.Fatalf("count platform outbox event: %v", err)
	}
	return count
}

func clearPlatformOutboxEvents(t *testing.T, pool interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM platform_outbox_events`); err != nil {
		t.Fatalf("clear Platform outbox events: %v", err)
	}
}

func clearPlatformConsumerDeduplications(t *testing.T, pool interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM platform_consumer_deduplications`); err != nil {
		t.Fatalf("clear Platform consumer deduplications: %v", err)
	}
}
