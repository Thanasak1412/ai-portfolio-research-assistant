//go:build integration

package database

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/infrastructure/database/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAssetConstraintsAndDeterministicBootstrap(t *testing.T) {
	pool := openAssetTestPool(t)
	queries := sqlcgen.New(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	suffix := strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))

	equity := bootstrapAsset(t, queries, "EQ"+suffix, "Equity "+suffix, "EQUITY", "NASDAQ", now)
	if equity.Currency != "USD" || equity.AssetType != "EQUITY" {
		t.Fatalf("unexpected canonical Asset: currency=%q type=%q", equity.Currency, equity.AssetType)
	}
	etf := bootstrapAsset(t, queries, "ET"+suffix, "ETF "+suffix, "ETF", "NYSEARCA", now)
	crypto := bootstrapAsset(t, queries, "CR"+suffix, "Crypto "+suffix, "CRYPTO", "CRYPTO", now)
	if crypto.Exchange != "CRYPTO" || !crypto.NormalizedExchange.Valid || crypto.NormalizedExchange.String != "CRYPTO" {
		t.Fatalf("crypto namespace = exchange %q normalized %#v", crypto.Exchange, crypto.NormalizedExchange)
	}

	repeated, err := queries.BootstrapCanonicalAsset(ctx, sqlcgen.BootstrapCanonicalAssetParams{
		AssetID:   newAssetUUID(),
		Symbol:    "eq" + strings.ToLower(suffix),
		Name:      "Updated Equity " + suffix,
		AssetType: "EQUITY",
		Exchange:  "nasdaq",
		CreatedAt: assetTime(now.Add(time.Minute)),
		UpdatedAt: assetTime(now.Add(time.Minute)),
	})
	if err != nil {
		t.Fatalf("repeat catalog bootstrap: %v", err)
	}
	if repeated.AssetID.Bytes != equity.AssetID.Bytes {
		t.Fatal("repeat catalog bootstrap changed canonical Asset ID")
	}
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM assets
		WHERE normalized_symbol = $1 AND normalized_exchange = $2`,
		equity.NormalizedSymbol.String, equity.NormalizedExchange.String,
	).Scan(&count); err != nil || count != 1 {
		t.Fatalf("catalog bootstrap identity count=%d err=%v", count, err)
	}

	got, err := queries.GetCanonicalAssetByID(ctx, etf.AssetID)
	if err != nil || got.AssetID.Bytes != etf.AssetID.Bytes {
		t.Fatalf("get canonical Asset: row=%v err=%v", got.AssetID, err)
	}

	assertAssetStatementRejected(t, pool, `
		INSERT INTO assets (
			asset_id, symbol, name, asset_type, exchange, currency, created_at, updated_at
		) VALUES ($1, $2, 'Unsupported', 'BOND', 'NYSE', 'USD', $3, $3)`,
		newAssetUUID(), "BD"+suffix, now,
	)
	assertAssetStatementRejected(t, pool, `
		INSERT INTO assets (
			asset_id, symbol, name, asset_type, exchange, currency, created_at, updated_at
		) VALUES ($1, $2, 'Wrong Currency', 'EQUITY', 'NYSE', 'THB', $3, $3)`,
		newAssetUUID(), "CU"+suffix, now,
	)
	assertAssetStatementRejected(t, pool, `
		INSERT INTO assets (
			asset_id, symbol, name, asset_type, exchange, currency, created_at, updated_at
		) VALUES ($1, $2, 'Duplicate', 'EQUITY', 'NASDAQ', 'USD', $3, $3)`,
		newAssetUUID(), strings.ToLower(equity.Symbol), now,
	)
	assertAssetStatementRejected(t, pool, `
		INSERT INTO assets (
			asset_id, symbol, name, asset_type, exchange, currency, created_at, updated_at
		) VALUES ($1, $2, 'Wrong Crypto Venue', 'CRYPTO', 'COINBASE', 'USD', $3, $3)`,
		newAssetUUID(), "VC"+suffix, now,
	)

	assertAssetColumnsAbsent(t, pool, "assets",
		"owner_user_id", "portfolio_id", "quantity", "average_cost", "cost_basis",
		"purchase_price", "market_value", "realized_pnl", "unrealized_pnl",
		"allocation", "current_price", "provider_price_id",
	)
}

func TestCanonicalAssetSearchIsCaseInsensitiveStableAndKeysetCompatible(t *testing.T) {
	pool := openAssetTestPool(t)
	queries := sqlcgen.New(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	suffix := strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))

	first := bootstrapAsset(t, queries, "AA"+suffix, "Atlas "+suffix, "EQUITY", "NYSE", now)
	second := bootstrapAsset(t, queries, "BB"+suffix, "Beacon "+suffix, "ETF", "NASDAQ", now.Add(time.Second))
	third := bootstrapAsset(t, queries, "CC"+suffix, "Cobalt "+suffix, "CRYPTO", "CRYPTO", now.Add(2*time.Second))

	bySymbol, err := queries.SearchCanonicalAssets(ctx, searchParams(strings.ToLower("AA"+suffix), "", "", "", pgtype.UUID{}, 10))
	if err != nil || len(bySymbol) != 1 || bySymbol[0].AssetID.Bytes != first.AssetID.Bytes {
		t.Fatalf("case-insensitive symbol search: rows=%d err=%v", len(bySymbol), err)
	}
	byName, err := queries.SearchCanonicalAssets(ctx, searchParams(strings.ToLower("BEACON "+suffix), "", "", "", pgtype.UUID{}, 10))
	if err != nil || len(byName) != 1 || byName[0].AssetID.Bytes != second.AssetID.Bytes {
		t.Fatalf("case-insensitive display-name search: rows=%d err=%v", len(byName), err)
	}
	etfs, err := queries.SearchCanonicalAssets(ctx, searchParams(suffix, "ETF", "", "", pgtype.UUID{}, 10))
	if err != nil || len(etfs) != 1 || etfs[0].AssetID.Bytes != second.AssetID.Bytes {
		t.Fatalf("exact AssetType filter: rows=%d err=%v", len(etfs), err)
	}

	firstPage, err := queries.SearchCanonicalAssets(ctx, searchParams(suffix, "", "", "", pgtype.UUID{}, 2))
	if err != nil || len(firstPage) != 2 {
		t.Fatalf("first cursor page: rows=%d err=%v", len(firstPage), err)
	}
	if firstPage[0].AssetID.Bytes != first.AssetID.Bytes || firstPage[1].AssetID.Bytes != second.AssetID.Bytes {
		t.Fatalf("stable order = [%v %v], want [%v %v]", firstPage[0].Symbol, firstPage[1].Symbol, first.Symbol, second.Symbol)
	}
	cursor := firstPage[1]
	secondPage, err := queries.SearchCanonicalAssets(ctx, searchParams(
		suffix, "", cursor.Symbol, cursor.Exchange, cursor.AssetID, 2,
	))
	if err != nil || len(secondPage) != 1 || secondPage[0].AssetID.Bytes != third.AssetID.Bytes {
		t.Fatalf("keyset continuation: rows=%d err=%v", len(secondPage), err)
	}
}

func TestAssetPersistenceIndexesSupportCanonicalLookupAndPublicOrdering(t *testing.T) {
	pool := openAssetTestPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin query-plan transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SET LOCAL enable_seqscan = off"); err != nil {
		t.Fatalf("disable sequential scans for query-plan assertion: %v", err)
	}
	explain := func(statement string) string {
		rows, err := tx.Query(ctx, statement)
		if err != nil {
			t.Fatalf("explain query plan: %v", err)
		}
		defer rows.Close()
		var plan strings.Builder
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				t.Fatalf("scan query plan: %v", err)
			}
			plan.WriteString(line)
			plan.WriteByte('\n')
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("read query plan: %v", err)
		}
		return plan.String()
	}

	identityPlan := explain(`
		EXPLAIN (FORMAT TEXT)
		SELECT asset_id
		FROM assets
		WHERE normalized_symbol = 'M2_DB_INDEX_PROBE'
		  AND normalized_exchange = 'NASDAQ'`)
	if !strings.Contains(identityPlan, "assets_normalized_identity_unique") {
		t.Fatalf("canonical identity lookup plan = %q, want unique identity index", identityPlan)
	}

	orderingPlan := explain(`
		EXPLAIN (FORMAT TEXT)
		SELECT asset_id, symbol, exchange
		FROM assets
		ORDER BY symbol ASC, exchange ASC, asset_id ASC
		LIMIT 25`)
	if !strings.Contains(orderingPlan, "assets_symbol_exchange_id_idx") {
		t.Fatalf("public Asset ordering plan = %q, want ordering index", orderingPlan)
	}
}

func openAssetTestPool(t *testing.T) *pgxpool.Pool {
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

func bootstrapAsset(t *testing.T, queries *sqlcgen.Queries, symbol, name, assetType, exchange string, now time.Time) sqlcgen.Asset {
	t.Helper()
	row, err := queries.BootstrapCanonicalAsset(context.Background(), sqlcgen.BootstrapCanonicalAssetParams{
		AssetID: newAssetUUID(), Symbol: symbol, Name: name, AssetType: assetType, Exchange: exchange,
		CreatedAt: assetTime(now), UpdatedAt: assetTime(now),
	})
	if err != nil {
		t.Fatalf("bootstrap Asset %s/%s: %v", symbol, exchange, err)
	}
	return row
}

func searchParams(search, assetType, cursorSymbol, cursorExchange string, cursorID pgtype.UUID, limit int32) sqlcgen.SearchCanonicalAssetsParams {
	params := sqlcgen.SearchCanonicalAssetsParams{PageLimit: limit, CursorAssetID: cursorID}
	if search != "" {
		params.Search = pgtype.Text{String: search, Valid: true}
	}
	if assetType != "" {
		params.AssetType = pgtype.Text{String: assetType, Valid: true}
	}
	if cursorSymbol != "" {
		params.CursorSymbol = pgtype.Text{String: cursorSymbol, Valid: true}
		params.CursorExchange = pgtype.Text{String: cursorExchange, Valid: true}
	}
	return params
}

func assertAssetStatementRejected(t *testing.T, pool *pgxpool.Pool, statement string, arguments ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), statement, arguments...); err == nil {
		t.Fatalf("expected statement to be rejected: %s", statement)
	}
}

func assertAssetColumnsAbsent(t *testing.T, pool *pgxpool.Pool, table string, forbidden ...string) {
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

func newAssetUUID() pgtype.UUID {
	value := uuid.New()
	return pgtype.UUID{Bytes: value, Valid: true}
}

func assetTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
