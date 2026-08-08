//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

func TestAuthenticationAuditParticipatesInIdentityTransaction(t *testing.T) {
	pool := openTestPool(t)
	transactor := NewPostgresTransactor(pool)
	correlationID := "corr-audit-transaction-" + uuid.NewString()
	event := application.AuditEvent{
		OccurredAt: time.Now().UTC(), Action: application.AuditRefreshFailure,
		Result: application.AuditResultFailure, Severity: application.AuditSeverityWarning,
		CorrelationID: correlationID,
	}
	rollbackCause := errors.New("force audit rollback")
	err := transactor.WithinTransaction(context.Background(), func(ctx context.Context, repositories application.TransactionRepositories) error {
		if err := repositories.Audit().Append(ctx, event); err != nil {
			return err
		}
		return rollbackCause
	})
	if !errors.Is(err, rollbackCause) {
		t.Fatalf("expected rollback cause, got %v", err)
	}
	if count := auditCount(t, pool, correlationID); count != 0 {
		t.Fatalf("rolled-back audit record persisted: count=%d", count)
	}

	if err := transactor.WithinTransaction(context.Background(), func(ctx context.Context, repositories application.TransactionRepositories) error {
		return repositories.Audit().Append(ctx, event)
	}); err != nil {
		t.Fatalf("commit audit evidence: %v", err)
	}
	if count := auditCount(t, pool, correlationID); count != 1 {
		t.Fatalf("committed audit record count=%d", count)
	}
}

func auditCount(t *testing.T, pool *pgxpool.Pool, correlationID string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM audit_logs WHERE correlation_id = $1", correlationID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
