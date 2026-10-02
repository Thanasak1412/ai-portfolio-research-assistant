package worker

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
)

// Publisher must return nil only after delivery succeeds. Receivers must tolerate
// duplicate delivery: a process may crash between Publish and MarkPublished.
type Publisher interface {
	Publish(context.Context, outbox.Event) error
}

type DeliveryConfig struct {
	LeaseDuration          time.Duration
	RetryBaseDelay         time.Duration
	RetryMaxDelay          time.Duration
	MaxDeliveryInvocations int32
	BatchSize              int32
	PollInterval           time.Duration
}

// Platform persists attempt_count as int32. Reserve one representable claim
// for administrative recovery after the last publisher invocation.
const MaximumDeliveryInvocations int32 = 2147483646

func DefaultDeliveryConfig() DeliveryConfig {
	return DeliveryConfig{60 * time.Second, 5 * time.Second, 5 * time.Minute, 10, 50, 2 * time.Second}
}

var ErrInvalidDeliveryConfig = errors.New("invalid outbox delivery configuration")
var ErrInvalidDeliveryDependencies = errors.New("invalid outbox delivery dependencies")

func (c DeliveryConfig) Validate() error {
	if c.LeaseDuration <= 0 || c.RetryBaseDelay <= 0 || c.RetryMaxDelay < c.RetryBaseDelay || c.MaxDeliveryInvocations < 1 || c.MaxDeliveryInvocations > MaximumDeliveryInvocations || c.BatchSize < 1 || c.BatchSize > outbox.MaximumClaimBatchSize || c.PollInterval <= 0 {
		return ErrInvalidDeliveryConfig
	}
	return nil
}

// DeliveryDependencies makes timing and randomness deterministic in tests. Wait
// must honor cancellation; Jitter returns an inclusive duration in [0, cap].
type DeliveryDependencies struct {
	Store      outbox.DeliveryStore
	Publisher  Publisher
	Now        func() time.Time
	Jitter     func(time.Duration) time.Duration
	Wait       func(context.Context, time.Duration) error
	ClaimToken func() ([16]byte, error)
}

type Delivery struct {
	config       DeliveryConfig
	dependencies DeliveryDependencies
}

func NewDelivery(config DeliveryConfig, dependencies DeliveryDependencies) (*Delivery, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if dependencies.Store == nil || dependencies.Publisher == nil {
		return nil, ErrInvalidDeliveryDependencies
	}
	if dependencies.Now == nil {
		dependencies.Now = time.Now
	}
	if dependencies.Jitter == nil {
		dependencies.Jitter = func(cap time.Duration) time.Duration {
			// Unsigned arithmetic preserves the inclusive endpoint even at MaxInt64.
			return time.Duration(rand.Uint64N(uint64(cap) + 1))
		}
	}
	if dependencies.Wait == nil {
		dependencies.Wait = waitDelivery
	}
	if dependencies.ClaimToken == nil {
		dependencies.ClaimToken = func() ([16]byte, error) { id, err := uuid.NewRandom(); return id, err }
	}
	return &Delivery{config, dependencies}, nil
}

func waitDelivery(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Run waits between every claim cycle, including a full batch or zero jitter.
// Persistence errors terminate this runner so its supervisor can report them;
// claims are never acknowledged speculatively on shutdown or error.
func (delivery *Delivery) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := delivery.RunOnce(ctx); err != nil {
			return err
		}
		if err := delivery.dependencies.Wait(ctx, delivery.config.PollInterval); err != nil {
			return err
		}
	}
}

func (delivery *Delivery) RunOnce(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := delivery.dependencies.Now().UTC()
	token, err := delivery.dependencies.ClaimToken()
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	claims, err := delivery.dependencies.Store.ClaimDue(ctx, outbox.ClaimRequest{
		AsOf: now, ClaimToken: token, LeaseExpiresAt: now.Add(delivery.config.LeaseDuration), BatchLimit: delivery.config.BatchSize,
	})
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if err := ctx.Err(); err != nil {
			return err
		}
		// A sequential batch may consume its lease before reaching this item.
		// Do not start delivery of an already expired claim.
		if !delivery.dependencies.Now().Before(claim.LeaseExpiresAt) {
			continue
		}
		if err := delivery.deliver(ctx, claim); err != nil {
			return err
		}
	}
	return nil
}

func (delivery *Delivery) deliver(ctx context.Context, claim outbox.ClaimedEvent) error {
	if claim.AttemptCount < 1 {
		return outbox.ErrInvalidClaimRequest
	}
	if claim.AttemptCount > delivery.config.MaxDeliveryInvocations {
		_, err := delivery.dependencies.Store.MarkDeadLetter(ctx, claim.ID, claim.ClaimToken, "delivery_attempts_exhausted")
		return err
	}
	// Bound cooperative publishing to the lease. A publisher must honor context
	// cancellation. Claim-token predicates still protect every durable transition.
	publishCtx, cancel := context.WithDeadline(ctx, claim.LeaseExpiresAt)
	err := delivery.dependencies.Publisher.Publish(publishCtx, claim.Event)
	cancel()
	if err == nil {
		// A successful in-flight delivery may be acknowledged during cancellation,
		// but only with this claim's token and a bounded independent context.
		ackCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), delivery.config.LeaseDuration)
		defer stop()
		_, err = delivery.dependencies.Store.MarkPublished(ackCtx, claim.ID, claim.ClaimToken, delivery.dependencies.Now().UTC())
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !delivery.dependencies.Now().Before(claim.LeaseExpiresAt) {
		return nil
	}
	if claim.AttemptCount == delivery.config.MaxDeliveryInvocations {
		_, err = delivery.dependencies.Store.MarkDeadLetter(ctx, claim.ID, claim.ClaimToken, "delivery_attempts_exhausted")
		return err
	}
	cap := delivery.retryCap(claim.AttemptCount)
	delay := delivery.dependencies.Jitter(cap)
	if delay < 0 || delay > cap {
		return ErrInvalidDeliveryDependencies
	}
	_, err = delivery.dependencies.Store.Reschedule(ctx, claim.ID, claim.ClaimToken, delivery.dependencies.Now().UTC().Add(delay), "delivery_failed")
	return err
}

func (delivery *Delivery) retryCap(attempt int32) time.Duration {
	cap := delivery.config.RetryBaseDelay
	for n := int32(1); n < attempt && cap < delivery.config.RetryMaxDelay; n++ {
		if cap > delivery.config.RetryMaxDelay/2 {
			return delivery.config.RetryMaxDelay
		}
		cap *= 2
	}
	return cap
}
