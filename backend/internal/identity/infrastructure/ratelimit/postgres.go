package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	platformdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
)

type PolicyDefinition struct {
	Limit  int64
	Window time.Duration
}

var policyDefinitions = map[application.RateLimitPolicy]PolicyDefinition{
	application.RateLimitLoginEmailFailure:     {Limit: 5, Window: 15 * time.Minute},
	application.RateLimitLoginIPAttempt:        {Limit: 30, Window: 15 * time.Minute},
	application.RateLimitRegistrationIPAttempt: {Limit: 5, Window: time.Hour},
	application.RateLimitRefreshFamilyAttempt:  {Limit: 20, Window: 15 * time.Minute},
}

type Store interface {
	WithinTransaction(context.Context, func(platformdatabase.AuthenticationRateLimitTransactionOperations) error) error
	CleanupExpired(context.Context, time.Time) (int64, error)
}

type PostgresLimiter struct{ store Store }

func NewPostgresLimiter(store Store) *PostgresLimiter { return &PostgresLimiter{store: store} }

func (limiter *PostgresLimiter) Check(
	ctx context.Context,
	policy application.RateLimitPolicy,
	key application.RateLimitKey,
	now time.Time,
) (application.RateLimitResult, error) {
	definition, exists := policyDefinitions[policy]
	if !exists || key.IsZero() || now.IsZero() {
		return application.RateLimitResult{}, application.ErrInvalidRateLimitKey
	}
	if limiter == nil || limiter.store == nil {
		return application.RateLimitResult{}, application.ErrRateLimitUnavailable
	}
	derivedKey := key.Bytes()
	result := application.RateLimitResult{}
	err := limiter.store.WithinTransaction(ctx, func(transaction platformdatabase.AuthenticationRateLimitTransactionOperations) error {
		if err := transaction.AcquireAdvisoryLock(ctx, advisoryLockKey(policy, derivedKey)); err != nil {
			return err
		}
		if _, err := transaction.DeleteExpiredForKey(ctx, string(policy), application.RateLimitPolicyVersion, derivedKey, now); err != nil {
			return err
		}
		count, err := transaction.CountActive(ctx, string(policy), application.RateLimitPolicyVersion, derivedKey, now)
		if err != nil {
			return err
		}
		if count >= definition.Limit {
			earliest, expiryErr := transaction.EarliestActiveExpiry(ctx, string(policy), application.RateLimitPolicyVersion, derivedKey, now)
			if expiryErr != nil {
				return expiryErr
			}
			result = application.RateLimitResult{Allowed: false, Remaining: 0, RetryAfter: maxDuration(earliest.Sub(now), time.Second)}
			return nil
		}
		if err := transaction.Insert(ctx, platformdatabase.AuthenticationRateLimitEvent{
			PolicyName: string(policy), PolicyVersion: application.RateLimitPolicyVersion,
			DerivedKey: derivedKey, OccurredAt: now, ExpiresAt: now.Add(definition.Window),
		}); err != nil {
			return err
		}
		result = application.RateLimitResult{Allowed: true, Remaining: int(definition.Limit - count - 1)}
		return nil
	})
	if err != nil {
		return application.RateLimitResult{}, safeStoreError(err)
	}
	return result, nil
}

func (limiter *PostgresLimiter) ResetLoginEmailFailures(ctx context.Context, key application.RateLimitKey) error {
	if key.IsZero() {
		return application.ErrInvalidRateLimitKey
	}
	if limiter == nil || limiter.store == nil {
		return application.ErrRateLimitUnavailable
	}
	derivedKey := key.Bytes()
	err := limiter.store.WithinTransaction(ctx, func(transaction platformdatabase.AuthenticationRateLimitTransactionOperations) error {
		if err := transaction.AcquireAdvisoryLock(ctx, advisoryLockKey(application.RateLimitLoginEmailFailure, derivedKey)); err != nil {
			return err
		}
		_, err := transaction.ClearLoginEmailFailures(ctx, application.RateLimitPolicyVersion, derivedKey)
		return err
	})
	return safeStoreError(err)
}

func (limiter *PostgresLimiter) CleanupExpired(ctx context.Context, now time.Time) (int64, error) {
	if limiter == nil || limiter.store == nil || now.IsZero() {
		return 0, application.ErrRateLimitUnavailable
	}
	count, err := limiter.store.CleanupExpired(ctx, now)
	if err != nil {
		return 0, safeStoreError(err)
	}
	return count, nil
}

func advisoryLockKey(policy application.RateLimitPolicy, key []byte) int64 {
	digest := sha256.Sum256(append([]byte(string(policy)+":"+application.RateLimitPolicyVersion+":"), key...))
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

func safeStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return errors.Join(application.ErrRateLimitUnavailable, errors.New("rate limit persistence operation failed"))
}

func maxDuration(left time.Duration, right time.Duration) time.Duration {
	if left > right {
		return left
	}
	return right
}

var _ application.RateLimiter = (*PostgresLimiter)(nil)
