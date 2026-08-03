//go:build integration

package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/config"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database/sqlcgen"
)

func TestDisposablePostgresIsReachable(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required for integration tests")
	}

	applicationConfig, err := config.Load(func(key string) (string, bool) {
		if key == "DATABASE_URL" {
			return databaseURL, true
		}
		return os.LookupEnv(key)
	})
	if err != nil {
		t.Fatalf("load test configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := Open(ctx, applicationConfig)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer pool.Close()
	if err := Verify(ctx, pool, applicationConfig.DatabaseConnectTimeout); err != nil {
		t.Fatalf("verify test database: %v", err)
	}
	value, err := sqlcgen.New(pool).DatabaseHealth(ctx)
	if err != nil {
		t.Fatalf("run generated database health query: %v", err)
	}
	if value != 1 {
		t.Fatalf("expected generated health query to return 1, got %d", value)
	}
}
