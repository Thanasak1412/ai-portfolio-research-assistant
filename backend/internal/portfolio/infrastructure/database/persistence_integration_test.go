//go:build integration

package database

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/infrastructure/database/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPortfolioPersistenceConstraintsAndOwnershipQueries(t *testing.T) {
	pool := openPortfolioTestPool(t)
	queries := sqlcgen.New(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	owner := insertPortfolioTestUser(t, pool, now)

	created := createPortfolio(t, queries, owner, "Growth", now)
	if created.Status != "ACTIVE" || created.ArchivedAt.Valid {
		t.Fatalf("unexpected active lifecycle: status=%q archived=%v", created.Status, created.ArchivedAt.Valid)
	}
	if !created.NormalizedName.Valid || created.NormalizedName.String != "growth" {
		t.Fatalf("normalized name = %#v, want growth", created.NormalizedName)
	}
	if created.BaseCurrency != "USD" {
		t.Fatalf("base currency = %q, want USD", created.BaseCurrency)
	}

	if _, err := queries.CreatePortfolio(ctx, sqlcgen.CreatePortfolioParams{
		PortfolioID: newPortfolioUUID(), OwnerUserID: owner, Name: "growth", CreatedAt: portfolioTime(now),
	}); !isUniqueViolation(err) {
		t.Fatalf("case-equivalent active name error = %v, want unique violation", err)
	}

	otherOwner := insertPortfolioTestUser(t, pool, now)
	if _, err := queries.CreatePortfolio(ctx, sqlcgen.CreatePortfolioParams{
		PortfolioID: newPortfolioUUID(), OwnerUserID: otherOwner, Name: "growth", CreatedAt: portfolioTime(now),
	}); err != nil {
		t.Fatalf("same normalized name for another owner: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO portfolios (
			portfolio_id, owner_user_id, name, base_currency, status, archived_at, created_at, updated_at
		) VALUES ($1, $2, $3, 'USD', 'ARCHIVED', $4, $5, $5)`,
		newPortfolioUUID(), owner, "Released Name", now, now,
	); err != nil {
		t.Fatalf("insert archived Portfolio: %v", err)
	}
	if _, err := queries.CreatePortfolio(ctx, sqlcgen.CreatePortfolioParams{
		PortfolioID: newPortfolioUUID(), OwnerUserID: owner, Name: "released name", CreatedAt: portfolioTime(now.Add(time.Second)),
	}); err != nil {
		t.Fatalf("archived Portfolio did not release name: %v", err)
	}

	archiveCandidate := createPortfolio(t, queries, owner, "Archive Then Reuse", now.Add(2*time.Second))
	if _, err := queries.ArchiveOwnedActivePortfolio(ctx, sqlcgen.ArchiveOwnedActivePortfolioParams{
		ArchivedAt:  portfolioTime(now.Add(3 * time.Second)),
		UpdatedAt:   portfolioTime(now.Add(3 * time.Second)),
		PortfolioID: archiveCandidate.PortfolioID,
		OwnerUserID: owner,
	}); err != nil {
		t.Fatalf("archive active Portfolio: %v", err)
	}
	if _, err := queries.CreatePortfolio(ctx, sqlcgen.CreatePortfolioParams{
		PortfolioID: newPortfolioUUID(), OwnerUserID: owner, Name: "archive then reuse", CreatedAt: portfolioTime(now.Add(4 * time.Second)),
	}); err != nil {
		t.Fatalf("archive did not release active-name uniqueness: %v", err)
	}

	renameTarget := createPortfolio(t, queries, owner, "Rename Target", now.Add(5*time.Second))
	renameConflict := createPortfolio(t, queries, owner, "Rename Conflict", now.Add(6*time.Second))
	if _, err := queries.UpdateOwnedActivePortfolioName(ctx, sqlcgen.UpdateOwnedActivePortfolioNameParams{
		Name: "rename conflict", UpdatedAt: portfolioTime(now.Add(7 * time.Second)),
		PortfolioID: renameTarget.PortfolioID, OwnerUserID: owner,
	}); !isUniqueViolation(err) {
		t.Fatalf("duplicate rename error = %v, want unique violation", err)
	}
	if _, err := queries.UpdateOwnedActivePortfolioName(ctx, sqlcgen.UpdateOwnedActivePortfolioNameParams{
		Name: "not allowed", UpdatedAt: portfolioTime(now.Add(7 * time.Second)),
		PortfolioID: renameConflict.PortfolioID, OwnerUserID: otherOwner,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("cross-owner update error = %v, want no rows", err)
	}
	if _, err := queries.UpdateOwnedActivePortfolioName(ctx, sqlcgen.UpdateOwnedActivePortfolioNameParams{
		Name: "not allowed", UpdatedAt: portfolioTime(now.Add(7 * time.Second)),
		PortfolioID: archiveCandidate.PortfolioID, OwnerUserID: owner,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("archived update error = %v, want no rows", err)
	}

	got, err := queries.GetOwnedPortfolioByID(ctx, sqlcgen.GetOwnedPortfolioByIDParams{
		PortfolioID: created.PortfolioID, OwnerUserID: owner,
	})
	if err != nil || got.PortfolioID.Bytes != created.PortfolioID.Bytes {
		t.Fatalf("get owned Portfolio: row=%v err=%v", got.PortfolioID, err)
	}
	if _, err := queries.GetOwnedPortfolioByID(ctx, sqlcgen.GetOwnedPortfolioByIDParams{
		PortfolioID: created.PortfolioID, OwnerUserID: otherOwner,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("cross-owner get error = %v, want no rows", err)
	}
	if _, err := queries.ArchiveOwnedActivePortfolio(ctx, sqlcgen.ArchiveOwnedActivePortfolioParams{
		ArchivedAt:  portfolioTime(now.Add(8 * time.Second)),
		UpdatedAt:   portfolioTime(now.Add(8 * time.Second)),
		PortfolioID: created.PortfolioID,
		OwnerUserID: otherOwner,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("cross-owner archive error = %v, want no rows", err)
	}

	listed, err := queries.ListOwnedPortfoliosByStatus(ctx, sqlcgen.ListOwnedPortfoliosByStatusParams{
		OwnerUserID: owner, Status: "ACTIVE",
	})
	if err != nil {
		t.Fatalf("list owner active Portfolios: %v", err)
	}
	for _, row := range listed {
		if row.OwnerUserID.Bytes != owner.Bytes || row.Status != "ACTIVE" {
			t.Fatalf("owner-scoped list leaked row: owner=%v status=%q", row.OwnerUserID, row.Status)
		}
	}

	assertPortfolioStatementRejected(t, pool, `
		INSERT INTO portfolios (
			portfolio_id, owner_user_id, name, base_currency, status, archived_at, created_at, updated_at
		) VALUES ($1, $2, 'Invalid Status', 'USD', 'PENDING', NULL, $3, $3)`,
		newPortfolioUUID(), owner, now,
	)
	assertPortfolioStatementRejected(t, pool, `
		INSERT INTO portfolios (
			portfolio_id, owner_user_id, name, base_currency, status, archived_at, created_at, updated_at
		) VALUES ($1, $2, 'Invalid Currency', 'THB', 'ACTIVE', NULL, $3, $3)`,
		newPortfolioUUID(), owner, now,
	)
	assertPortfolioStatementRejected(t, pool, `
		INSERT INTO portfolios (
			portfolio_id, owner_user_id, name, base_currency, status, archived_at, created_at, updated_at
		) VALUES ($1, $2, 'Active Archived', 'USD', 'ACTIVE', $3, $3, $3)`,
		newPortfolioUUID(), owner, now,
	)
	assertPortfolioStatementRejected(t, pool, `
		INSERT INTO portfolios (
			portfolio_id, owner_user_id, name, base_currency, status, archived_at, created_at, updated_at
		) VALUES ($1, $2, 'Archived Missing Timestamp', 'USD', 'ARCHIVED', NULL, $3, $3)`,
		newPortfolioUUID(), owner, now,
	)
	assertPortfolioStatementRejected(t, pool, `
		INSERT INTO portfolios (
			portfolio_id, owner_user_id, name, base_currency, status, archived_at, created_at, updated_at
		) VALUES ($1, $2, 'Unknown Owner', 'USD', 'ACTIVE', NULL, $3, $3)`,
		newPortfolioUUID(), newPortfolioUUID(), now,
	)
	assertPortfolioStatementRejected(t, pool, `
		INSERT INTO portfolios (
			portfolio_id, owner_user_id, name, base_currency, status, archived_at, created_at, updated_at
		) VALUES ($1, $2, ' Leading Space', 'USD', 'ACTIVE', NULL, $3, $3)`,
		newPortfolioUUID(), owner, now,
	)

	assertPortfolioColumnsAbsent(t, pool, "portfolios",
		"quantity", "average_cost", "cost_basis", "purchase_price", "market_value",
		"realized_pnl", "unrealized_pnl", "allocation", "current_price", "asset_id",
	)
	if _, exists := reflect.TypeOf(queries).MethodByName("DeletePortfolio"); exists {
		t.Fatal("Portfolio sqlc surface must not expose hard delete")
	}
}

func TestPortfolioActiveNameUniquenessIsConcurrent(t *testing.T) {
	pool := openPortfolioTestPool(t)
	secondPool := openPortfolioTestPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	owner := insertPortfolioTestUser(t, pool, now)
	name := "Concurrent " + uuid.NewString()

	start := make(chan struct{})
	results := make(chan error, 2)
	querySets := []*sqlcgen.Queries{sqlcgen.New(pool), sqlcgen.New(secondPool)}
	for i := range 2 {
		i := i
		go func() {
			<-start
			_, err := querySets[i].CreatePortfolio(ctx, sqlcgen.CreatePortfolioParams{
				PortfolioID: newPortfolioUUID(),
				OwnerUserID: owner,
				Name:        name,
				CreatedAt:   portfolioTime(now),
			})
			results <- err
		}()
	}
	close(start)

	successes := 0
	uniqueViolations := 0
	for range 2 {
		err := <-results
		if err == nil {
			successes++
		} else if isUniqueViolation(err) {
			uniqueViolations++
		} else {
			t.Fatalf("concurrent create error = %v", err)
		}
	}
	if successes != 1 || uniqueViolations != 1 {
		t.Fatalf("concurrent active-name results: successes=%d uniqueness=%d", successes, uniqueViolations)
	}
}

func TestPortfolioPersistenceIndexSupportsOwnerScopedStableListing(t *testing.T) {
	pool := openPortfolioTestPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin query-plan transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SET LOCAL enable_seqscan = off"); err != nil {
		t.Fatalf("disable sequential scans for query-plan assertion: %v", err)
	}

	var plan string
	if err := tx.QueryRow(ctx, `
		EXPLAIN (FORMAT TEXT)
		SELECT portfolio_id
		FROM portfolios
		WHERE owner_user_id = '00000000-0000-0000-0000-000000000001'
		  AND status = 'ACTIVE'
		ORDER BY updated_at DESC, portfolio_id ASC`).Scan(&plan); err != nil {
		t.Fatalf("explain owner-scoped Portfolio listing: %v", err)
	}
	if !strings.Contains(plan, "portfolios_owner_status_updated_id_idx") {
		t.Fatalf("owner-scoped Portfolio listing plan = %q, want listing index", plan)
	}
}

func openPortfolioTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func insertPortfolioTestUser(t *testing.T, pool *pgxpool.Pool, now time.Time) pgtype.UUID {
	t.Helper()
	id := newPortfolioUUID()
	email := "portfolio-db-" + uuid.NewString() + "@example.test"
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO users (
			user_id, normalized_email, password_hash, account_status, created_at, updated_at, disabled_at
		) VALUES ($1, $2, $3, 'active', $4, $4, NULL)`,
		id, email, "$argon2id$v=19$m=1,t=1,p=1$fixture$fixture", now,
	); err != nil {
		t.Fatalf("insert test owner: %v", err)
	}
	return id
}

func createPortfolio(t *testing.T, queries *sqlcgen.Queries, owner pgtype.UUID, name string, now time.Time) sqlcgen.Portfolio {
	t.Helper()
	row, err := queries.CreatePortfolio(context.Background(), sqlcgen.CreatePortfolioParams{
		PortfolioID: newPortfolioUUID(), OwnerUserID: owner, Name: name, CreatedAt: portfolioTime(now),
	})
	if err != nil {
		t.Fatalf("create Portfolio %q: %v", name, err)
	}
	return row
}

func assertPortfolioStatementRejected(t *testing.T, pool *pgxpool.Pool, statement string, arguments ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), statement, arguments...); err == nil {
		t.Fatalf("expected statement to be rejected: %s", statement)
	}
}

func assertPortfolioColumnsAbsent(t *testing.T, pool *pgxpool.Pool, table string, forbidden ...string) {
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

func newPortfolioUUID() pgtype.UUID {
	value := uuid.New()
	return pgtype.UUID{Bytes: value, Valid: true}
}

func portfolioTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
