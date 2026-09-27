//go:build integration

package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ctx = context.Background()
var fixtureTime = time.Date(2026, 8, 1, 12, 0, 0, 123456000, time.UTC)

// Every test owns a fresh disposable schema. Two independently configured pools
// share that schema for cross-instance tests; no existing test/business rows are
// cleared. This also allows the other modules' suites to execute concurrently.
func testPools(t *testing.T, version int) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required for PostgreSQL integration")
	}
	config, err := pgxpool.ParseConfig(dsn)
	must(t, err)
	if !strings.HasSuffix(config.ConnConfig.Database, "_test") {
		t.Fatal("integration requires an explicitly named _test database")
	}
	admin, err := pgxpool.NewWithConfig(ctx, config)
	must(t, err)
	schema := "m3db_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize())
	must(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		if err != nil {
			t.Errorf("drop isolated test schema: %v", err)
		}
		admin.Close()
	})
	open := func() *pgxpool.Pool {
		c := config.Copy()
		c.ConnConfig.RuntimeParams["search_path"] = schema
		p, err := pgxpool.NewWithConfig(ctx, c)
		must(t, err)
		t.Cleanup(p.Close)
		return p
	}
	first, second := open(), open()
	migrations, err := filepath.Glob("../../../../migrations/[0-9]*.sql")
	must(t, err)
	if len(migrations) != 5 {
		t.Fatalf("expected five migrations, got %d", len(migrations))
	}
	for _, file := range migrations[:version] {
		runMigration(t, first, file, true)
	}
	return first, second
}

func migrationSQL(t *testing.T, file string, up bool) string {
	t.Helper()
	data, err := os.ReadFile(file)
	must(t, err)
	parts := strings.Split(string(data), "-- +goose Down")
	if len(parts) != 2 {
		t.Fatalf("invalid migration %s", file)
	}
	if up {
		return parts[0]
	}
	return parts[1]
}
func runMigration(t *testing.T, pool *pgxpool.Pool, file string, up bool) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	must(t, err)
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, migrationSQL(t, file, up))
	must(t, err)
	must(t, tx.Commit(ctx))
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func uid() pgtype.UUID                       { return pgtype.UUID{Bytes: uuid.New(), Valid: true} }
func instant(v time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: v, Valid: true} }
func txt(v string) pgtype.Text               { return pgtype.Text{String: v, Valid: true} }
func num(v string) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(v); err != nil {
		panic(err)
	}
	return n
}

type fixture struct{ owner, portfolio, asset pgtype.UUID }

func seed(t *testing.T, pool *pgxpool.Pool) fixture {
	t.Helper()
	f := fixture{uid(), uid(), uid()}
	// Synthetic noncredential hash metadata; never a usable login fixture.
	_, err := pool.Exec(ctx, `INSERT INTO users(user_id,normalized_email,password_hash,account_status,created_at,updated_at)
        VALUES($1,$2,'persistence-test-not-a-credential','active',$3,$3)`, f.owner, uuid.NewString()+"@example.test", fixtureTime)
	must(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO portfolios(portfolio_id,owner_user_id,name,base_currency,status,created_at,updated_at)
        VALUES($1,$2,'Persistence fixture','USD','ACTIVE',$3,$3)`, f.portfolio, f.owner, fixtureTime)
	must(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO assets(asset_id,symbol,name,asset_type,exchange,currency,created_at,updated_at)
        VALUES($1,$2,'Synthetic fixture','EQUITY','NASDAQ','USD',$3,$3)`, f.asset, uuid.NewString(), fixtureTime)
	must(t, err)
	return f
}
func command(f fixture, kind string, sequence int64) sqlcgen.InsertTransactionParams {
	p := sqlcgen.InsertTransactionParams{TransactionID: uid(), PortfolioID: f.portfolio, CreatedByUserID: f.owner, Kind: kind, Currency: "USD", PortfolioSequence: sequence, EffectiveAt: instant(fixtureTime), CreatedAt: instant(fixtureTime)}
	if kind == "BUY" || kind == "SELL" || kind == "DIVIDEND" {
		p.AssetID = f.asset
		p.AssetTypeSnapshot = txt("EQUITY")
		p.AssetExchangeSnapshot = txt("NASDAQ")
		p.AssetCurrencySnapshot = txt("USD")
	}
	if kind == "BUY" || kind == "SELL" {
		p.Quantity = num("2.125")
		p.UnitPrice = num("10.25")
		p.Fee = num("0")
	} else {
		p.Amount = num("25")
	}
	return p
}
func insert(t *testing.T, q *sqlcgen.Queries, p sqlcgen.InsertTransactionParams) sqlcgen.Transaction {
	t.Helper()
	row, err := q.InsertTransaction(ctx, p)
	must(t, err)
	return row
}
