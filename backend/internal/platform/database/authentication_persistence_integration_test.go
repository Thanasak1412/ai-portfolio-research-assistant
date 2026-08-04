//go:build integration

package database

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthenticationAuditPersistenceIsAppendOnlyAndBounded(t *testing.T) {
	pool := openAuthenticationTestPool(t)
	queries := sqlcgen.New(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	event, err := queries.AppendAuthenticationAuditEvent(ctx, sqlcgen.AppendAuthenticationAuditEventParams{
		AuditEventID: platformUUID(), OccurredAt: platformTime(now), Action: "login_failure",
		Result: "failure", Severity: "warning", CorrelationID: "corr-" + uuid.NewString(),
		NetworkIdentityHash: pgtype.Text{String: "hmac:v1:network-fixture", Valid: true},
		UserAgent:           pgtype.Text{String: "integration-test", Valid: true},
	})
	if err != nil {
		t.Fatalf("append Authentication audit event: %v", err)
	}
	if event.Action != "login_failure" || event.ActorUserID.Valid {
		t.Fatalf("unexpected audit event: action=%q actor_valid=%v", event.Action, event.ActorUserID.Valid)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_logs (audit_event_id, occurred_at, action, result, severity, correlation_id)
		VALUES ($1, $2, 'credential_details', 'failure', 'warning', $3)`,
		platformUUID(), now, "corr-"+uuid.NewString()); err == nil {
		t.Fatal("expected unapproved audit action rejection")
	}
	assertPlatformColumnsAbsent(t, pool, "audit_logs",
		"password", "password_length", "password_hash", "access_token", "refresh_token",
		"cookie", "authorization_header", "private_key", "metadata", "payload")
	querySurface := reflect.TypeOf(queries)
	for _, forbiddenMethod := range []string{"UpdateAuthenticationAuditEvent", "DeleteAuthenticationAuditEvent"} {
		if _, exists := querySurface.MethodByName(forbiddenMethod); exists {
			t.Fatalf("append-only audit query surface exposes %s", forbiddenMethod)
		}
	}
}

func TestAuthenticationRateLimitPersistenceAndIsolation(t *testing.T) {
	pool := openAuthenticationTestPool(t)
	queries := sqlcgen.New(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	emailKey := platformDigest(1)
	// Reuse the same derived bytes under a different namespace to prove namespace isolation.
	ipKey := emailKey
	familyKey := platformDigest(3)

	insertRateEvent(t, queries, "login_email_failure", emailKey, now, now.Add(15*time.Minute))
	insertRateEvent(t, queries, "login_email_failure", emailKey, now.Add(time.Second), now.Add(10*time.Minute))
	insertRateEvent(t, queries, "login_ip_attempt", ipKey, now, now.Add(15*time.Minute))
	insertRateEvent(t, queries, "refresh_family_attempt", familyKey, now, now.Add(15*time.Minute))
	insertRateEvent(t, queries, "registration_ip_attempt", platformDigest(4), now.Add(-2*time.Hour), now.Add(-time.Hour))

	count, err := queries.CountActiveAuthRateLimitEvents(ctx, sqlcgen.CountActiveAuthRateLimitEventsParams{
		PolicyName: "login_email_failure", PolicyVersion: "v1", DerivedKey: emailKey, AsOf: platformTime(now),
	})
	if err != nil || count != 2 {
		t.Fatalf("count active email events: count=%d err=%v", count, err)
	}
	earliest, err := queries.GetEarliestActiveAuthRateLimitExpiry(ctx, sqlcgen.GetEarliestActiveAuthRateLimitExpiryParams{
		PolicyName: "login_email_failure", PolicyVersion: "v1", DerivedKey: emailKey, AsOf: platformTime(now),
	})
	if err != nil || !earliest.Valid || !earliest.Time.Equal(now.Add(10*time.Minute)) {
		t.Fatalf("earliest active expiry: value=%v err=%v", earliest, err)
	}
	cleared, err := queries.ClearLoginEmailFailureEvents(ctx, sqlcgen.ClearLoginEmailFailureEventsParams{
		PolicyVersion: "v1", DerivedKey: emailKey,
	})
	if err != nil || cleared != 2 {
		t.Fatalf("clear email failures: rows=%d err=%v", cleared, err)
	}
	ipCount, err := queries.CountActiveAuthRateLimitEvents(ctx, sqlcgen.CountActiveAuthRateLimitEventsParams{
		PolicyName: "login_ip_attempt", PolicyVersion: "v1", DerivedKey: ipKey, AsOf: platformTime(now),
	})
	if err != nil || ipCount != 1 {
		t.Fatalf("email reset changed IP policy: count=%d err=%v", ipCount, err)
	}
	familyCount, err := queries.CountActiveAuthRateLimitEvents(ctx, sqlcgen.CountActiveAuthRateLimitEventsParams{
		PolicyName: "refresh_family_attempt", PolicyVersion: "v1", DerivedKey: familyKey, AsOf: platformTime(now),
	})
	if err != nil || familyCount != 1 {
		t.Fatalf("refresh family isolation: count=%d err=%v", familyCount, err)
	}
	deleted, err := queries.DeleteGloballyExpiredAuthRateLimitEvents(ctx, platformTime(now))
	if err != nil || deleted < 1 {
		t.Fatalf("global expiry cleanup: rows=%d err=%v", deleted, err)
	}
	assertPlatformColumnsAbsent(t, pool, "auth_rate_limit_events",
		"email", "ip", "ip_address", "refresh_token", "cookie", "credential", "authorization_header")
}

func TestAuthenticationRateLimitAdvisoryLockSerializesConnections(t *testing.T) {
	pool := openAuthenticationTestPool(t)
	secondPool := openAuthenticationTestPool(t)
	ctx := context.Background()
	lockKey := int64(8_174_901)

	txOne, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin first transaction: %v", err)
	}
	if err := sqlcgen.New(pool).WithTx(txOne).AcquireAuthRateLimitAdvisoryLock(ctx, lockKey); err != nil {
		t.Fatalf("acquire first advisory lock: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		txTwo, beginErr := secondPool.Begin(ctx)
		if beginErr != nil {
			result <- beginErr
			return
		}
		defer txTwo.Rollback(ctx)
		result <- sqlcgen.New(secondPool).WithTx(txTwo).AcquireAuthRateLimitAdvisoryLock(ctx, lockKey)
	}()
	select {
	case err := <-result:
		t.Fatalf("second advisory lock completed before release: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := txOne.Commit(ctx); err != nil {
		t.Fatalf("commit first transaction: %v", err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("second advisory lock after release: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second advisory lock did not resume")
	}
}

func openAuthenticationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func insertRateEvent(t *testing.T, queries *sqlcgen.Queries, policy string, key []byte, occurredAt, expiresAt time.Time) {
	t.Helper()
	if _, err := queries.InsertAuthRateLimitEvent(context.Background(), sqlcgen.InsertAuthRateLimitEventParams{
		DerivedKey: key, PolicyName: policy, PolicyVersion: "v1",
		OccurredAt: platformTime(occurredAt), ExpiresAt: platformTime(expiresAt),
	}); err != nil {
		t.Fatalf("insert %s rate-limit event: %v", policy, err)
	}
}

func assertPlatformColumnsAbsent(t *testing.T, pool *pgxpool.Pool, table string, forbidden ...string) {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = $1`, table)
	if err != nil {
		t.Fatalf("inspect %s columns: %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("scan %s column: %v", table, err)
		}
		for _, disallowed := range forbidden {
			if strings.EqualFold(column, disallowed) {
				t.Fatalf("table %s contains forbidden column %s", table, column)
			}
		}
	}
}

func platformUUID() pgtype.UUID {
	value := uuid.New()
	return pgtype.UUID{Bytes: value, Valid: true}
}

func platformTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func platformDigest(seed byte) []byte {
	value := uuid.New()
	result := make([]byte, 32)
	copy(result, value[:])
	copy(result[16:], value[:])
	result[31] ^= seed
	return result
}
