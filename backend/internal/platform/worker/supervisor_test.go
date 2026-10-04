package worker

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
)

type checkedHeartbeat func(context.Context) error

func (f checkedHeartbeat) Ping(ctx context.Context) error { return f(ctx) }

// Any unexpected persistence operation panics through the embedded nil store.
// ClaimDue additionally counts calls to make the activation invariant explicit.
type inactiveStore struct {
	outbox.DeliveryStore
	claims int
}

func (s *inactiveStore) ClaimDue(context.Context, outbox.ClaimRequest) ([]outbox.ClaimedEvent, error) {
	s.claims++
	return nil, errors.New("inactive composition must not claim")
}

func TestRunConfiguredWithoutReceiverNeverStartsDelivery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var logs bytes.Buffer
	store := &inactiveStore{}
	beats := 0
	checker := checkedHeartbeat(func(context.Context) error {
		beats++
		if beats == 3 {
			cancel()
		}
		return nil
	})
	err := RunConfigured(ctx, slog.New(slog.NewTextHandler(&logs, nil)), checker, time.Nanosecond, DefaultDeliveryConfig(), DeliveryDependencies{
		Store:      store,
		ClaimToken: func() ([16]byte, error) { t.Fatal("claim token allocated while inactive"); return [16]byte{}, nil },
		Jitter:     func(time.Duration) time.Duration { t.Fatal("retry scheduled while inactive"); return 0 },
	})
	if !errors.Is(err, context.Canceled) || beats < 3 || store.claims != 0 {
		t.Fatalf("inactive lifecycle: err=%v beats=%d claims=%d", err, beats, store.claims)
	}
	if !strings.Contains(logs.String(), "transaction publication inactive") || !strings.Contains(logs.String(), "no_approved_receiver") {
		t.Fatal("inactive publication was not reported")
	}
}

func TestRunConfiguredDoesNotTreatInvalidActiveDependenciesAsInactive(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	err := RunConfigured(context.Background(), logger, checkerStub{}, time.Second, DefaultDeliveryConfig(), DeliveryDependencies{
		Publisher: publisherFunc(func(context.Context, outbox.Event) error { t.Fatal("invalid configuration published"); return nil }),
	})
	if !errors.Is(err, ErrInvalidDeliveryDependencies) {
		t.Fatal("missing active store silently disabled delivery", err)
	}
}

func TestRunConfiguredWithReceiverUsesDeliveryEngine(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	accepted := 0
	delivery, store, _ := deliveryFixture(t, 1, func(context.Context, outbox.Event) error {
		accepted++
		cancel()
		return nil
	})
	err := RunConfigured(ctx, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), checkerStub{}, time.Hour, DefaultDeliveryConfig(), delivery.dependencies)
	if !errors.Is(err, context.Canceled) || accepted != 1 || store.claimCalls != 1 || store.published != 1 || store.rescheduled != 0 || store.dead != 0 {
		t.Fatalf("active composition did not preserve successful in-flight handoff: %v", err)
	}
}

func TestInactiveCompositionStillRejectsInvalidPolicy(t *testing.T) {
	config := DefaultDeliveryConfig()
	config.BatchSize = 0
	err := RunConfigured(context.Background(), slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), checkerStub{}, time.Second, config, DeliveryDependencies{})
	if !errors.Is(err, ErrInvalidDeliveryConfig) {
		t.Fatal("inactive composition bypassed configuration validation", err)
	}
}
