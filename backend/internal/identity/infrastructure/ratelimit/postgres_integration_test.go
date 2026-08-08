//go:build integration

package ratelimit

import (
	"context"
	"crypto/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	platformdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
)

func TestPostgresLimiterSerializesAcrossTwoPools(t *testing.T) {
	firstPool := openRateLimitPool(t)
	secondPool := openRateLimitPool(t)
	limiters := []*PostgresLimiter{
		NewPostgresLimiter(platformdatabase.NewAuthenticationRateLimitStore(firstPool)),
		NewPostgresLimiter(platformdatabase.NewAuthenticationRateLimitStore(secondPool)),
	}
	keyBytes := make([]byte, application.RateLimitKeyLength)
	if _, err := rand.Read(keyBytes); err != nil {
		t.Fatal(err)
	}
	key, _ := application.NewRateLimitKey(keyBytes)
	now := time.Now().UTC().Truncate(time.Microsecond)

	const attempts = 20
	var allowed atomic.Int64
	var failures atomic.Int64
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := 0; index < attempts; index++ {
		wait.Add(1)
		go func(limiter *PostgresLimiter) {
			defer wait.Done()
			<-start
			result, err := limiter.Check(context.Background(), application.RateLimitRegistrationIPAttempt, key, now)
			if err != nil {
				failures.Add(1)
				return
			}
			if result.Allowed {
				allowed.Add(1)
			}
		}(limiters[index%len(limiters)])
	}
	close(start)
	wait.Wait()
	if failures.Load() != 0 || allowed.Load() != 5 {
		t.Fatalf("cross-pool limit: allowed=%d failures=%d", allowed.Load(), failures.Load())
	}
}

func openRateLimitPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required for integration tests")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
