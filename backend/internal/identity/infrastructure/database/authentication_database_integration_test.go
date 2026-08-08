//go:build integration

package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/database/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUsersPersistenceAndConstraints(t *testing.T) {
	pool := openTestPool(t)
	queries := sqlcgen.New(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := newPGUUID()
	email := uniqueEmail("identity")

	created, err := queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		UserID: userID, NormalizedEmail: email, PasswordHash: encodedHash("initial"), CreatedAt: pgTime(now),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.AccountStatus != "active" || created.NormalizedEmail != email {
		t.Fatalf("unexpected created user: status=%q email=%q", created.AccountStatus, created.NormalizedEmail)
	}
	byEmail, err := queries.GetUserByNormalizedEmail(ctx, email)
	if err != nil || byEmail.UserID.Bytes != userID.Bytes {
		t.Fatalf("get user by normalized email: user=%v err=%v", byEmail.UserID, err)
	}
	if _, err := queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		UserID: newPGUUID(), NormalizedEmail: email, PasswordHash: encodedHash("duplicate"), CreatedAt: pgTime(now),
	}); err == nil {
		t.Fatal("expected normalized-email uniqueness violation")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (user_id, normalized_email, password_hash, account_status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', $4, $4)`, newPGUUID(), " Not.Normalized@Example.com ", encodedHash("bad-email"), now); err == nil {
		t.Fatal("expected non-normalized email rejection")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (user_id, normalized_email, password_hash, account_status, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', $4, $4)`, newPGUUID(), uniqueEmail("bad-status"), encodedHash("bad-status"), now); err == nil {
		t.Fatal("expected invalid account-status rejection")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (user_id, normalized_email, password_hash, account_status, created_at, updated_at)
		VALUES ($1, $2, '', 'active', $3, $3)`, newPGUUID(), uniqueEmail("blank-hash"), now); err == nil {
		t.Fatal("expected blank password-hash rejection")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE users SET account_status = 'disabled' WHERE user_id = $1`, userID); err == nil {
		t.Fatal("expected disabled status without disabled_at to be rejected")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE users SET account_status = 'disabled', disabled_at = $2, updated_at = $2 WHERE user_id = $1`, userID, now.Add(time.Second)); err != nil {
		t.Fatalf("persist valid disabled state: %v", err)
	}

	oldHash := encodedHash("initial")
	newHash := encodedHash("rehash")
	updated, err := queries.UpdatePasswordHashCompareAndSwap(ctx, sqlcgen.UpdatePasswordHashCompareAndSwapParams{
		NewPasswordHash: newHash, UpdatedAt: pgTime(now.Add(2 * time.Second)), UserID: userID, ExpectedPasswordHash: oldHash,
	})
	if err != nil || updated.PasswordHash != newHash {
		t.Fatalf("compare-and-swap password hash: hash_match=%v err=%v", updated.PasswordHash == newHash, err)
	}
	if _, err := queries.UpdatePasswordHashCompareAndSwap(ctx, sqlcgen.UpdatePasswordHashCompareAndSwapParams{
		NewPasswordHash: encodedHash("stale"), UpdatedAt: pgTime(now.Add(3 * time.Second)), UserID: userID, ExpectedPasswordHash: oldHash,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected stale compare-and-swap to affect no row, got %v", err)
	}

	assertColumnsAbsent(t, pool, "users", "password", "raw_password", "access_token", "refresh_token")
}

func TestRefreshSessionConstraintsAndFamilyIsolation(t *testing.T) {
	pool := openTestPool(t)
	queries := sqlcgen.New(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := insertTestUser(t, queries, now)
	familyOne := newPGUUID()
	first := insertSession(t, queries, userID, familyOne, digest(1), now)

	locked, err := beginAndReadSession(t, pool, first.TokenDigest)
	if err != nil {
		t.Fatalf("read session by digest for update: %v", err)
	}
	if err := locked.Rollback(ctx); err != nil {
		t.Fatalf("rollback read transaction: %v", err)
	}
	if _, err := queries.InsertRefreshSessionGeneration(ctx, sessionParams(userID, newPGUUID(), first.TokenDigest, now.Add(time.Second))); err == nil {
		t.Fatal("expected duplicate token digest rejection")
	}
	if _, err := queries.InsertRefreshSessionGeneration(ctx, sessionParams(newPGUUID(), newPGUUID(), digest(2), now)); err == nil {
		t.Fatal("expected user foreign-key rejection")
	}
	if _, err := queries.InsertRefreshSessionGeneration(ctx, sessionParams(userID, familyOne, digest(3), now.Add(time.Second))); err == nil {
		t.Fatal("expected one-active-generation-per-family rejection")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO refresh_sessions (
			session_id, token_family_id, user_id, token_digest, session_state,
			created_at, idle_expires_at, absolute_expires_at
		) VALUES ($1, $2, $3, $4, 'active', $5, $6, $7)`,
		newPGUUID(), newPGUUID(), userID, digest(4), now, now.Add(91*24*time.Hour), now.Add(90*24*time.Hour)); err == nil {
		t.Fatal("expected idle expiry beyond absolute expiry rejection")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO refresh_sessions (
			session_id, token_family_id, user_id, token_digest, session_state,
			created_at, idle_expires_at, absolute_expires_at
		) VALUES ($1, $2, $3, $4, 'replaced', $5, $6, $7)`,
		newPGUUID(), newPGUUID(), userID, digest(5), now, now.Add(30*24*time.Hour), now.Add(90*24*time.Hour)); err == nil {
		t.Fatal("expected invalid replaced lifecycle rejection")
	}

	familyTwo := newPGUUID()
	second := insertSession(t, queries, userID, familyTwo, digest(6), now)
	revoked, err := queries.RevokeRefreshTokenFamily(ctx, sqlcgen.RevokeRefreshTokenFamilyParams{
		RevokedAt:        pgTime(now.Add(time.Minute)),
		RevocationReason: pgtype.Text{String: "reuse_detected", Valid: true},
		TokenFamilyID:    familyOne,
	})
	if err != nil || len(revoked) != 1 || revoked[0].SessionState != "revoked" {
		t.Fatalf("revoke requested family: rows=%d err=%v", len(revoked), err)
	}
	unrelated, err := queries.GetRefreshSessionByID(ctx, second.SessionID)
	if err != nil || unrelated.SessionState != "active" {
		t.Fatalf("unrelated family changed: state=%q err=%v", unrelated.SessionState, err)
	}
	assertColumnsAbsent(t, pool, "refresh_sessions", "refresh_token", "raw_token", "cookie", "access_token", "authorization_header", "private_key", "password")
}

func TestRefreshRotationPrimitivesSerializeAcrossConnections(t *testing.T) {
	pool := openTestPool(t)
	secondPool := openTestPool(t)
	queries := sqlcgen.New(pool)
	secondQueries := sqlcgen.New(secondPool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := insertTestUser(t, queries, now)
	familyID := newPGUUID()
	original := insertSession(t, queries, userID, familyID, digest(20), now)

	txOne, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin first transaction: %v", err)
	}
	qOne := queries.WithTx(txOne)
	if _, err := qOne.GetRefreshSessionByTokenDigestForUpdate(ctx, original.TokenDigest); err != nil {
		t.Fatalf("lock original generation: %v", err)
	}

	lockResult := make(chan error, 1)
	go func() {
		txTwo, beginErr := secondPool.Begin(ctx)
		if beginErr != nil {
			lockResult <- beginErr
			return
		}
		defer txTwo.Rollback(ctx)
		_, lockErr := secondQueries.WithTx(txTwo).GetRefreshSessionByTokenDigestForUpdate(ctx, original.TokenDigest)
		lockResult <- lockErr
	}()
	select {
	case err := <-lockResult:
		t.Fatalf("competing FOR UPDATE completed before release: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := txOne.Commit(ctx); err != nil {
		t.Fatalf("commit first lock holder: %v", err)
	}
	select {
	case err := <-lockResult:
		if err != nil {
			t.Fatalf("competing FOR UPDATE after release: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("competing FOR UPDATE did not resume after release")
	}

	var successes int
	var mu sync.Mutex
	start := make(chan struct{})
	errorsCh := make(chan error, 2)
	pools := []*pgxpool.Pool{pool, secondPool}
	querySets := []*sqlcgen.Queries{queries, secondQueries}
	for attempt := byte(0); attempt < 2; attempt++ {
		attempt := attempt
		go func() {
			<-start
			tx, beginErr := pools[attempt].Begin(ctx)
			if beginErr != nil {
				errorsCh <- beginErr
				return
			}
			defer tx.Rollback(ctx)
			qtx := querySets[attempt].WithTx(tx)
			row, lockErr := qtx.GetRefreshSessionByTokenDigestForUpdate(ctx, original.TokenDigest)
			if lockErr != nil {
				errorsCh <- lockErr
				return
			}
			if row.SessionState != "active" {
				errorsCh <- nil
				return
			}
			replacementID := newPGUUID()
			rotatedAt := now.Add(time.Duration(attempt+1) * time.Minute)
			if _, updateErr := qtx.MarkActiveRefreshSessionReplaced(ctx, sqlcgen.MarkActiveRefreshSessionReplacedParams{
				ReplacementSessionID: replacementID, ReplacedAt: pgTime(rotatedAt), SessionID: original.SessionID,
			}); updateErr != nil {
				errorsCh <- updateErr
				return
			}
			params := sessionParams(userID, familyID, digest(30+attempt), rotatedAt)
			params.SessionID = replacementID
			params.AbsoluteExpiresAt = original.AbsoluteExpiresAt
			if _, insertErr := qtx.InsertRefreshSessionGeneration(ctx, params); insertErr != nil {
				errorsCh <- insertErr
				return
			}
			if commitErr := tx.Commit(ctx); commitErr != nil {
				errorsCh <- commitErr
				return
			}
			mu.Lock()
			successes++
			mu.Unlock()
			errorsCh <- nil
		}()
	}
	close(start)
	for range 2 {
		if err := <-errorsCh; err != nil {
			t.Fatalf("concurrent rotation primitive: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one successful rotation, got %d", successes)
	}
	family, err := queries.ListRefreshTokenFamilyState(ctx, familyID)
	if err != nil {
		t.Fatalf("list family: %v", err)
	}
	active := 0
	for _, generation := range family {
		if generation.SessionState == "active" {
			active++
		}
	}
	if len(family) != 2 || active != 1 {
		t.Fatalf("expected two generations and one active, got generations=%d active=%d", len(family), active)
	}
}

func TestRefreshRotationRollbackLeavesOriginalActive(t *testing.T) {
	pool := openTestPool(t)
	queries := sqlcgen.New(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := insertTestUser(t, queries, now)
	familyID := newPGUUID()
	original := insertSession(t, queries, userID, familyID, digest(50), now)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	qtx := queries.WithTx(tx)
	replacementID := newPGUUID()
	if _, err := qtx.GetRefreshSessionByTokenDigestForUpdate(ctx, original.TokenDigest); err != nil {
		t.Fatalf("lock original: %v", err)
	}
	if _, err := qtx.MarkActiveRefreshSessionReplaced(ctx, sqlcgen.MarkActiveRefreshSessionReplacedParams{
		ReplacementSessionID: replacementID, ReplacedAt: pgTime(now.Add(time.Minute)), SessionID: original.SessionID,
	}); err != nil {
		t.Fatalf("mark replaced: %v", err)
	}
	params := sessionParams(userID, familyID, digest(51), now.Add(time.Minute))
	params.SessionID = replacementID
	params.AbsoluteExpiresAt = original.AbsoluteExpiresAt
	if _, err := qtx.InsertRefreshSessionGeneration(ctx, params); err != nil {
		t.Fatalf("insert replacement: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	current, err := queries.GetRefreshSessionByID(ctx, original.SessionID)
	if err != nil || current.SessionState != "active" {
		t.Fatalf("original was changed after rollback: state=%q err=%v", current.SessionState, err)
	}
	if _, err := queries.GetRefreshSessionByID(ctx, replacementID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("replacement survived rollback: %v", err)
	}
}

func openTestPool(t *testing.T) *pgxpool.Pool {
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

func insertTestUser(t *testing.T, queries *sqlcgen.Queries, now time.Time) pgtype.UUID {
	t.Helper()
	userID := newPGUUID()
	if _, err := queries.CreateUser(context.Background(), sqlcgen.CreateUserParams{
		UserID: userID, NormalizedEmail: uniqueEmail("session"), PasswordHash: encodedHash("session"), CreatedAt: pgTime(now),
	}); err != nil {
		t.Fatalf("create session owner: %v", err)
	}
	return userID
}

func insertSession(t *testing.T, queries *sqlcgen.Queries, userID, familyID pgtype.UUID, tokenDigest []byte, createdAt time.Time) sqlcgen.RefreshSession {
	t.Helper()
	row, err := queries.InsertRefreshSessionGeneration(context.Background(), sessionParams(userID, familyID, tokenDigest, createdAt))
	if err != nil {
		t.Fatalf("insert refresh session: %v", err)
	}
	return row
}

func sessionParams(userID, familyID pgtype.UUID, tokenDigest []byte, createdAt time.Time) sqlcgen.InsertRefreshSessionGenerationParams {
	return sqlcgen.InsertRefreshSessionGenerationParams{
		SessionID: newPGUUID(), TokenFamilyID: familyID, UserID: userID, TokenDigest: tokenDigest,
		CreatedAt: pgTime(createdAt), IdleExpiresAt: pgTime(createdAt.Add(30 * 24 * time.Hour)),
		AbsoluteExpiresAt: pgTime(createdAt.Add(90 * 24 * time.Hour)),
	}
}

func beginAndReadSession(t *testing.T, pool *pgxpool.Pool, tokenDigest []byte) (pgx.Tx, error) {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	_, err = sqlcgen.New(pool).WithTx(tx).GetRefreshSessionByTokenDigestForUpdate(context.Background(), tokenDigest)
	return tx, err
}

func assertColumnsAbsent(t *testing.T, pool *pgxpool.Pool, table string, forbidden ...string) {
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

func newPGUUID() pgtype.UUID {
	value := uuid.New()
	return pgtype.UUID{Bytes: value, Valid: true}
}

func pgTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@example.test", prefix, uuid.NewString())
}

func encodedHash(label string) string {
	return "$argon2id$v=19$m=65536,t=3,p=1$" + label + "$fixture-not-a-credential"
}

func digest(seed byte) []byte {
	value := uuid.New()
	result := make([]byte, 32)
	copy(result, value[:])
	copy(result[16:], value[:])
	result[31] ^= seed
	return result
}
