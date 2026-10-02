package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
)

type publisherFunc func(context.Context, outbox.Event) error

func (f publisherFunc) Publish(ctx context.Context, e outbox.Event) error { return f(ctx, e) }

type deliveryStore struct {
	claims                                   []outbox.ClaimedEvent
	claimCalls, published, rescheduled, dead int
	code                                     string
	next                                     time.Time
	currentToken                             [16]byte
	request                                  outbox.ClaimRequest
}

func (*deliveryStore) Append(context.Context, outbox.Event) error { return nil }
func (s *deliveryStore) ClaimDue(_ context.Context, r outbox.ClaimRequest) ([]outbox.ClaimedEvent, error) {
	s.claimCalls++
	s.request = r
	return s.claims, nil
}
func (s *deliveryStore) MarkPublished(_ context.Context, _, token [16]byte, _ time.Time) (bool, error) {
	if token != s.currentToken {
		return false, nil
	}
	s.published++
	return true, nil
}
func (s *deliveryStore) Reschedule(_ context.Context, _, token [16]byte, next time.Time, code string) (bool, error) {
	if token != s.currentToken {
		return false, nil
	}
	s.rescheduled++
	s.next = next
	s.code = code
	return true, nil
}
func (s *deliveryStore) MarkDeadLetter(_ context.Context, _, token [16]byte, code string) (bool, error) {
	if token != s.currentToken {
		return false, nil
	}
	s.dead++
	s.code = code
	return true, nil
}

func deliveryFixture(t *testing.T, attempt int32, publish publisherFunc) (*Delivery, *deliveryStore, time.Time) {
	t.Helper()
	now := time.Now().UTC()
	token := [16]byte{1}
	store := &deliveryStore{currentToken: token, claims: []outbox.ClaimedEvent{{Event: outbox.Event{ID: [16]byte{2}}, AttemptCount: attempt, ClaimToken: token, LeaseExpiresAt: now.Add(time.Minute)}}}
	delivery, err := NewDelivery(DefaultDeliveryConfig(), DeliveryDependencies{
		Store: store, Publisher: publish, Now: func() time.Time { return now },
		ClaimToken: func() ([16]byte, error) { return token, nil },
		Jitter:     func(cap time.Duration) time.Duration { return cap },
	})
	if err != nil {
		t.Fatal(err)
	}
	return delivery, store, now
}

func TestDeliverySuccess(t *testing.T) {
	calls := 0
	d, s, _ := deliveryFixture(t, 1, func(context.Context, outbox.Event) error { calls++; return nil })
	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || s.published != 1 || s.rescheduled != 0 || s.dead != 0 {
		t.Fatal("expected one successful delivery")
	}
	if s.request.BatchLimit != 50 || s.request.LeaseExpiresAt.Sub(s.request.AsOf) != time.Minute {
		t.Fatal("ADR-022 claim settings")
	}
}

func TestDeliveryRetryCaps(t *testing.T) {
	expected := []time.Duration{5, 10, 20, 40, 80, 160, 300, 300, 300}
	for i, seconds := range expected {
		d, s, now := deliveryFixture(t, int32(i+1), func(context.Context, outbox.Event) error { return errors.New("private publisher failure") })
		if err := d.RunOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		if s.rescheduled != 1 || s.published != 0 || s.dead != 0 || s.next.Sub(now) != seconds*time.Second || s.code != "delivery_failed" {
			t.Fatalf("attempt %d retry mismatch", i+1)
		}
	}
}

func TestFinalInvocationAndAdministrativeRecovery(t *testing.T) {
	for _, attempt := range []int32{10, 11, 12} {
		calls := 0
		d, s, _ := deliveryFixture(t, attempt, func(context.Context, outbox.Event) error { calls++; return errors.New("retryable") })
		if err := d.RunOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		want := 0
		if attempt == 10 {
			want = 1
		}
		if calls != want || s.dead != 1 || s.rescheduled != 0 || s.published != 0 || s.code != "delivery_attempts_exhausted" {
			t.Fatalf("attempt %d must exhaust without extra invocation", attempt)
		}
	}
}

func TestStaleDeliveryCannotTransition(t *testing.T) {
	for _, scenario := range []string{"success", "retry", "exhausted"} {
		attempt := int32(1)
		if scenario == "exhausted" {
			attempt = 11
		}
		d, s, _ := deliveryFixture(t, attempt, func(context.Context, outbox.Event) error {
			if scenario == "success" {
				return nil
			}
			return errors.New("retryable")
		})
		s.currentToken = [16]byte{9}
		if err := d.RunOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		if s.published+s.rescheduled+s.dead != 0 {
			t.Fatal("stale worker changed durable state")
		}
	}
}

func TestCancellation(t *testing.T) {
	for _, success := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		d, s, _ := deliveryFixture(t, 1, func(context.Context, outbox.Event) error {
			cancel()
			if success {
				return nil
			}
			return context.Canceled
		})
		err := d.Run(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected cancellation: %v", err)
		}
		want := 0
		if success {
			want = 1
		}
		if s.claimCalls != 1 || s.published != want || s.rescheduled+s.dead != 0 {
			t.Fatal("shutdown transition mismatch")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d, s, _ := deliveryFixture(t, 1, func(context.Context, outbox.Event) error { t.Fatal("cancelled delivery"); return nil })
	if !errors.Is(d.RunOnce(ctx), context.Canceled) || s.claimCalls != 0 {
		t.Fatal("cancelled worker claimed")
	}
}

func TestZeroJitterStillWaitsBetweenClaims(t *testing.T) {
	d, s, now := deliveryFixture(t, 1, func(context.Context, outbox.Event) error { return errors.New("retryable") })
	d.dependencies.Jitter = func(time.Duration) time.Duration { return 0 }
	waits := 0
	d.dependencies.Wait = func(_ context.Context, duration time.Duration) error {
		waits++
		if duration != 2*time.Second {
			t.Fatal("incorrect poll interval")
		}
		return context.Canceled
	}
	if !errors.Is(d.Run(context.Background()), context.Canceled) || waits != 1 || s.claimCalls != 1 || !s.next.Equal(now) {
		t.Fatal("zero jitter bypassed poll pacing")
	}
}

func TestExpiredBatchItemIsNotPublished(t *testing.T) {
	d, s, now := deliveryFixture(t, 1, func(context.Context, outbox.Event) error { t.Fatal("expired claim published"); return nil })
	s.claims[0].LeaseExpiresAt = now
	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.published+s.dead+s.rescheduled != 0 {
		t.Fatal("expired claim transitioned")
	}
}

func TestRetryCapDoesNotOverflow(t *testing.T) {
	d, _, _ := deliveryFixture(t, 1, func(context.Context, outbox.Event) error { return nil })
	d.config.RetryBaseDelay = time.Duration(1 << 62)
	d.config.RetryMaxDelay = time.Duration(1<<63 - 1)
	if got := d.retryCap(2147483647); got != d.config.RetryMaxDelay {
		t.Fatalf("overflow: %v", got)
	}
}
