//go:build integration

package database

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	platform "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/worker"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
)

type inactiveHeartbeat func(context.Context) error

func (h inactiveHeartbeat) Ping(ctx context.Context) error { return h(ctx) }

type countingClaimStore struct {
	outbox.DeliveryStore
	claims int
}

func (s *countingClaimStore) ClaimDue(ctx context.Context, r outbox.ClaimRequest) ([]outbox.ClaimedEvent, error) {
	s.claims++
	return s.DeliveryStore.ClaimDue(ctx, r)
}

func TestCommittedTransactionRemainsPendingDuringInactiveWorker(t *testing.T) {
	pool, _ := testPools(t, 5)
	f := seed(t, pool)
	principal, pid := appOwner(f)
	_, err := appService(t, pool, nil).Create(ctx, principal, pid, appCommand(f, domain.KindBuy), appMeta())
	must(t, err)
	snapshot := func() string {
		var row string
		must(t, pool.QueryRow(ctx, "SELECT row_to_json(e)::text FROM platform_outbox_events e").Scan(&row))
		return row
	}
	before := snapshot()
	var state string
	var attempts int
	must(t, pool.QueryRow(ctx, "SELECT publication_state,attempt_count FROM platform_outbox_events").Scan(&state, &attempts))
	if state != "PENDING" || attempts != 0 {
		t.Fatal("command did not commit a pending event")
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	beats := 0
	checker := inactiveHeartbeat(func(c context.Context) error {
		if err := pool.Ping(c); err != nil {
			return err
		}
		beats++
		if beats == 3 {
			cancel()
		}
		return nil
	})
	store := &countingClaimStore{DeliveryStore: platform.NewPostgresOutboxStore(pool)}
	err = worker.RunConfigured(runCtx, slog.New(slog.NewTextHandler(io.Discard, nil)), checker, time.Nanosecond, worker.DefaultDeliveryConfig(), worker.DeliveryDependencies{Store: store})
	if !errors.Is(err, context.Canceled) || beats < 3 || store.claims != 0 {
		t.Fatalf("inactive composition: err=%v beats=%d claims=%d", err, beats, store.claims)
	}
	if after := snapshot(); after != before {
		t.Fatal("inactive worker changed durable event state or metadata")
	}
}
