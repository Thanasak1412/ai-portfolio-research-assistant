//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	platform "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/worker"
	"github.com/google/uuid"
)

type integrationPublisher func(context.Context, outbox.Event) error

func (p integrationPublisher) Publish(ctx context.Context, e outbox.Event) error { return p(ctx, e) }
func deliveryEvent(now time.Time, aggregate [16]byte) outbox.Event {
	return outbox.Event{ID: uuid.New(), Type: "transaction.recorded.v1", Version: 1, AggregateType: "portfolio", AggregateID: aggregate, PortfolioID: aggregate, OccurredAt: now, NextAttemptAt: now, CorrelationID: "delivery-integration", Payload: outbox.Payload{SchemaVersion: 1, References: []outbox.Reference{{Role: "transaction", ID: uuid.New()}}}}
}

func TestApplicationOutboxDeliveryAndFinalLeaseRecovery(t *testing.T) {
	pool, _ := testPools(t, 5)
	store := platform.NewPostgresOutboxStore(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	first := deliveryEvent(now, uuid.New())
	later := deliveryEvent(now, first.AggregateID)
	must(t, store.Append(ctx, first))
	must(t, store.Append(ctx, later))
	// Simulate ten crashed claim owners. Every claim is durable, even if the
	// external side effect is uncertain. The eleventh must be administrative.
	for i := 1; i <= 10; i++ {
		claimed, err := store.ClaimDue(ctx, outbox.ClaimRequest{AsOf: now, ClaimToken: uuid.New(), LeaseExpiresAt: now.Add(time.Minute), BatchLimit: 50})
		must(t, err)
		if len(claimed) != 1 || claimed[0].ID != first.ID || claimed[0].AttemptCount != int32(i) {
			t.Fatal("lease recovery or predecessor order")
		}
		now = now.Add(time.Minute)
	}
	calls := 0
	d, err := worker.NewDelivery(worker.DefaultDeliveryConfig(), worker.DeliveryDependencies{Store: store, Publisher: integrationPublisher(func(context.Context, outbox.Event) error { calls++; return nil }), Now: func() time.Time { return now }})
	must(t, err)
	must(t, d.RunOnce(ctx))
	must(t, d.RunOnce(ctx))
	if calls != 0 {
		t.Fatal("publisher called after final claim expiry")
	}
	var state, code string
	var attempts int
	must(t, pool.QueryRow(ctx, "SELECT publication_state,last_failure_code,attempt_count FROM platform_outbox_events WHERE event_id=$1", dbID(first.ID)).Scan(&state, &code, &attempts))
	if state != "DEAD_LETTER" || code != "delivery_attempts_exhausted" || attempts != 11 {
		t.Fatal("administrative recovery", state, code, attempts)
	}
	must(t, pool.QueryRow(ctx, "SELECT publication_state,attempt_count FROM platform_outbox_events WHERE event_id=$1", dbID(later.ID)).Scan(&state, &attempts))
	if state != "PENDING" || attempts != 0 {
		t.Fatal("dead-letter predecessor was bypassed")
	}
	other := deliveryEvent(now, uuid.New())
	must(t, store.Append(ctx, other))
	must(t, d.RunOnce(ctx))
	if calls != 1 {
		t.Fatal("independent stream blocked")
	}
	must(t, pool.QueryRow(ctx, "SELECT publication_state FROM platform_outbox_events WHERE event_id=$1", dbID(other.ID)).Scan(&state))
	if state != "PUBLISHED" {
		t.Fatal("publish not acknowledged")
	}
	must(t, d.RunOnce(ctx))
	if calls != 1 {
		t.Fatal("published event reclaimed")
	}
}

func TestApplicationOutboxRetriesAndStaleOwnership(t *testing.T) {
	pool, _ := testPools(t, 5)
	store := platform.NewPostgresOutboxStore(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	event := deliveryEvent(now, uuid.New())
	must(t, store.Append(ctx, event))
	calls := 0
	d, err := worker.NewDelivery(worker.DefaultDeliveryConfig(), worker.DeliveryDependencies{Store: store, Publisher: integrationPublisher(func(context.Context, outbox.Event) error { calls++; return errors.New("publisher unavailable") }), Now: func() time.Time { return now }, Jitter: func(time.Duration) time.Duration { return 0 }})
	must(t, err)
	for i := 1; i <= 10; i++ {
		must(t, d.RunOnce(ctx))
		var state string
		must(t, pool.QueryRow(ctx, "SELECT publication_state FROM platform_outbox_events WHERE event_id=$1", dbID(event.ID)).Scan(&state))
		want := "PENDING"
		if i == 10 {
			want = "DEAD_LETTER"
		}
		if state != want {
			t.Fatal("attempt state", i, state)
		}
	}
	must(t, d.RunOnce(ctx))
	if calls != 10 {
		t.Fatal("invocation limit", calls)
	}
	event = deliveryEvent(now, uuid.New())
	must(t, store.Append(ctx, event))
	oldToken, newToken := uuid.New(), uuid.New()
	claim, err := store.ClaimDue(ctx, outbox.ClaimRequest{AsOf: now, ClaimToken: oldToken, LeaseExpiresAt: now.Add(time.Minute), BatchLimit: 1})
	must(t, err)
	if len(claim) != 1 {
		t.Fatal("initial claim")
	}
	now = now.Add(time.Minute)
	claim, err = store.ClaimDue(ctx, outbox.ClaimRequest{AsOf: now, ClaimToken: newToken, LeaseExpiresAt: now.Add(time.Minute), BatchLimit: 1})
	must(t, err)
	if len(claim) != 1 || claim[0].AttemptCount != 2 {
		t.Fatal("reclaim")
	}
	for _, transition := range []func() (bool, error){
		func() (bool, error) { return store.MarkPublished(ctx, event.ID, oldToken, now) },
		func() (bool, error) { return store.Reschedule(ctx, event.ID, oldToken, now, "delivery_failed") },
		func() (bool, error) {
			return store.MarkDeadLetter(ctx, event.ID, oldToken, "delivery_attempts_exhausted")
		},
	} {
		changed, err := transition()
		must(t, err)
		if changed {
			t.Fatal("stale ownership accepted")
		}
	}
	changed, err := store.MarkPublished(ctx, event.ID, newToken, now)
	must(t, err)
	if !changed {
		t.Fatal("current owner rejected")
	}
}
