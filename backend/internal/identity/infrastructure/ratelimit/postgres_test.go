package ratelimit

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	platformdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
)

type memoryRateLimitStore struct {
	mu     sync.Mutex
	events []platformdatabase.AuthenticationRateLimitEvent
	err    error
}

func (store *memoryRateLimitStore) WithinTransaction(
	_ context.Context,
	operation func(platformdatabase.AuthenticationRateLimitTransactionOperations) error,
) error {
	if store.err != nil {
		return store.err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return operation((*memoryRateLimitTransaction)(store))
}

func (store *memoryRateLimitStore) CleanupExpired(_ context.Context, asOf time.Time) (int64, error) {
	if store.err != nil {
		return 0, store.err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	kept := store.events[:0]
	var removed int64
	for _, event := range store.events {
		if !event.ExpiresAt.After(asOf) {
			removed++
			continue
		}
		kept = append(kept, event)
	}
	store.events = kept
	return removed, nil
}

type memoryRateLimitTransaction memoryRateLimitStore

func (*memoryRateLimitTransaction) AcquireAdvisoryLock(context.Context, int64) error { return nil }

func (transaction *memoryRateLimitTransaction) DeleteExpiredForKey(_ context.Context, policy, version string, key []byte, asOf time.Time) (int64, error) {
	kept := transaction.events[:0]
	var removed int64
	for _, event := range transaction.events {
		if event.PolicyName == policy && event.PolicyVersion == version && string(event.DerivedKey) == string(key) && !event.ExpiresAt.After(asOf) {
			removed++
			continue
		}
		kept = append(kept, event)
	}
	transaction.events = kept
	return removed, nil
}

func (transaction *memoryRateLimitTransaction) CountActive(_ context.Context, policy, version string, key []byte, asOf time.Time) (int64, error) {
	var count int64
	for _, event := range transaction.events {
		if event.PolicyName == policy && event.PolicyVersion == version && string(event.DerivedKey) == string(key) && event.ExpiresAt.After(asOf) {
			count++
		}
	}
	return count, nil
}

func (transaction *memoryRateLimitTransaction) Insert(_ context.Context, event platformdatabase.AuthenticationRateLimitEvent) error {
	transaction.events = append(transaction.events, event)
	return nil
}

func (transaction *memoryRateLimitTransaction) EarliestActiveExpiry(_ context.Context, policy, version string, key []byte, asOf time.Time) (time.Time, error) {
	var earliest time.Time
	for _, event := range transaction.events {
		if event.PolicyName != policy || event.PolicyVersion != version || string(event.DerivedKey) != string(key) || !event.ExpiresAt.After(asOf) {
			continue
		}
		if earliest.IsZero() || event.ExpiresAt.Before(earliest) {
			earliest = event.ExpiresAt
		}
	}
	if earliest.IsZero() {
		return time.Time{}, errors.New("no active event")
	}
	return earliest, nil
}

func (transaction *memoryRateLimitTransaction) ClearLoginEmailFailures(_ context.Context, version string, key []byte) (int64, error) {
	kept := transaction.events[:0]
	var removed int64
	for _, event := range transaction.events {
		if event.PolicyName == string(application.RateLimitLoginEmailFailure) && event.PolicyVersion == version && string(event.DerivedKey) == string(key) {
			removed++
			continue
		}
		kept = append(kept, event)
	}
	transaction.events = kept
	return removed, nil
}

func TestPostgresLimiterExactPoliciesBoundariesAndRollingExpiry(t *testing.T) {
	definitions := map[application.RateLimitPolicy]PolicyDefinition{
		application.RateLimitLoginEmailFailure:     {Limit: 5, Window: 15 * time.Minute},
		application.RateLimitLoginIPAttempt:        {Limit: 30, Window: 15 * time.Minute},
		application.RateLimitRegistrationIPAttempt: {Limit: 5, Window: time.Hour},
		application.RateLimitRefreshFamilyAttempt:  {Limit: 20, Window: 15 * time.Minute},
	}
	for policy, definition := range definitions {
		t.Run(string(policy), func(t *testing.T) {
			store := &memoryRateLimitStore{}
			limiter := NewPostgresLimiter(store)
			key := testRateLimitKey(t, byte(definition.Limit))
			now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
			for attempt := int64(0); attempt < definition.Limit; attempt++ {
				result, err := limiter.Check(context.Background(), policy, key, now)
				if err != nil || !result.Allowed || result.Remaining != int(definition.Limit-attempt-1) {
					t.Fatalf("attempt %d: result=%+v err=%v", attempt, result, err)
				}
			}
			blocked, err := limiter.Check(context.Background(), policy, key, now)
			if err != nil || blocked.Allowed || blocked.RetryAfter != definition.Window {
				t.Fatalf("limit boundary: result=%+v err=%v", blocked, err)
			}
			afterWindow, err := limiter.Check(context.Background(), policy, key, now.Add(definition.Window))
			if err != nil || !afterWindow.Allowed {
				t.Fatalf("rolling expiry did not reopen policy: result=%+v err=%v", afterWindow, err)
			}
		})
	}
}

func TestPostgresLimiterResetIsolationCleanupAndSafeFailure(t *testing.T) {
	store := &memoryRateLimitStore{}
	limiter := NewPostgresLimiter(store)
	now := time.Now().UTC()
	sharedKey := testRateLimitKey(t, 7)
	otherKey := testRateLimitKey(t, 8)
	if _, err := limiter.Check(context.Background(), application.RateLimitLoginEmailFailure, sharedKey, now); err != nil {
		t.Fatal(err)
	}
	if _, err := limiter.Check(context.Background(), application.RateLimitLoginIPAttempt, sharedKey, now); err != nil {
		t.Fatal(err)
	}
	if _, err := limiter.Check(context.Background(), application.RateLimitLoginEmailFailure, otherKey, now); err != nil {
		t.Fatal(err)
	}
	if err := limiter.ResetLoginEmailFailures(context.Background(), sharedKey); err != nil {
		t.Fatal(err)
	}
	if len(store.events) != 2 {
		t.Fatalf("reset removed unrelated namespace or identity: events=%d", len(store.events))
	}
	removed, err := limiter.CleanupExpired(context.Background(), now.Add(16*time.Minute))
	if err != nil || removed != 2 {
		t.Fatalf("cleanup: removed=%d err=%v", removed, err)
	}

	secret := "database-secret-value"
	failing := NewPostgresLimiter(&memoryRateLimitStore{err: errors.New(secret)})
	if _, err := failing.Check(context.Background(), application.RateLimitLoginEmailFailure, sharedKey, now); !errors.Is(err, application.ErrRateLimitUnavailable) || strings.Contains(err.Error(), secret) {
		t.Fatalf("unsafe fail-closed result: %v", err)
	}
}

func testRateLimitKey(t *testing.T, seed byte) application.RateLimitKey {
	t.Helper()
	value := make([]byte, application.RateLimitKeyLength)
	for index := range value {
		value[index] = seed + byte(index)
	}
	key, err := application.NewRateLimitKey(value)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

var _ platformdatabase.AuthenticationRateLimitTransactionOperations = (*memoryRateLimitTransaction)(nil)
