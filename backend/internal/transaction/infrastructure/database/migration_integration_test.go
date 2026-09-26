//go:build integration

package database

import (
	"strings"
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database/sqlcgen"
)

const ledgerMigration = "../../../../migrations/00005_m3_transaction_ledger.sql"

func TestMigrationV4UpgradeEmptyDownUpAndIndexes(t *testing.T) {
	pool, _ := testPools(t, 4)
	f := seed(t, pool)
	runMigration(t, pool, ledgerMigration, true)
	q := sqlcgen.New(pool)
	rows, err := q.ListPortfolioTransactions(ctx, sqlcgen.ListPortfolioTransactionsParams{PortfolioID: f.portfolio, PageLimit: 50, IncludeReversals: true})
	must(t, err)
	if len(rows) != 0 {
		t.Fatal("migration created facts")
	}
	runMigration(t, pool, ledgerMigration, false)
	var exists bool
	must(t, pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM portfolios WHERE portfolio_id=$1)", f.portfolio).Scan(&exists))
	if !exists {
		t.Fatal("Down touched M2")
	}
	runMigration(t, pool, ledgerMigration, true)
	for _, name := range []string{"transactions_history_idx", "transactions_asset_replay_idx", "transactions_portfolio_sequence_unique", "transactions_direct_reversal_uidx", "transactions_direct_replacement_uidx", "corrections_one_original", "corrections_one_reversal", "corrections_one_replacement", "transaction_idempotency_pkey", "transaction_idempotency_expiry_idx"} {
		var definition string
		must(t, pool.QueryRow(ctx, "SELECT indexdef FROM pg_indexes WHERE schemaname=current_schema() AND indexname=$1", name).Scan(&definition))
		if !strings.Contains(definition, "CREATE") {
			t.Fatal("missing index")
		}
	}
	// Planner shape verification for both bounded history and replay indexes.
	tx, err := pool.Begin(ctx)
	must(t, err)
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "SET LOCAL enable_seqscan=off")
	must(t, err)
	for _, tc := range []struct{ sql, index string }{
		{"EXPLAIN (COSTS OFF) SELECT * FROM transactions WHERE portfolio_id=$1 ORDER BY effective_at DESC,portfolio_sequence DESC,transaction_id DESC LIMIT 50", "transactions_history_idx"},
		{"EXPLAIN (COSTS OFF) SELECT * FROM transactions WHERE portfolio_id=$1 AND asset_id=$2 ORDER BY effective_at,portfolio_sequence,transaction_id", "transactions_asset_replay_idx"},
	} {
		args := []any{f.portfolio}
		if strings.Contains(tc.sql, "$2") {
			args = append(args, f.asset)
		}
		plans, err := tx.Query(ctx, tc.sql, args...)
		must(t, err)
		var lines []string
		for plans.Next() {
			var line string
			must(t, plans.Scan(&line))
			lines = append(lines, line)
		}
		must(t, plans.Err())
		plans.Close()
		if !strings.Contains(strings.Join(lines, "\n"), tc.index) {
			t.Fatalf("missing intended index in plan: %v", lines)
		}
	}
}

func TestMigrationDownRefusesRetainedState(t *testing.T) {
	// Sequence-only durable state also prevents rollback, even without facts.
	for _, sequenceOnly := range []bool{true, false} {
		t.Run(map[bool]string{true: "sequence", false: "facts_and_idempotency"}[sequenceOnly], func(t *testing.T) {
			pool, _ := testPools(t, 5)
			f := seed(t, pool)
			q := sqlcgen.New(pool)
			if sequenceOnly {
				_, err := q.AllocateOnePortfolioSequence(ctx, f.portfolio)
				must(t, err)
			} else {
				p := command(f, "FEE", 1)
				insert(t, q, p)
				_, err := q.InsertCompletedTransactionIdempotency(ctx, completed(p, "rollback-proof-key"))
				must(t, err)
				tx, err := pool.Begin(ctx)
				must(t, err)
				correction(t, sqlcgen.New(tx), f, p, 2)
				must(t, tx.Commit(ctx))
			}
			tx, err := pool.Begin(ctx)
			must(t, err)
			_, err = tx.Exec(ctx, migrationSQL(t, ledgerMigration, false))
			if err == nil {
				t.Fatal("populated Down succeeded")
			}
			if !strings.Contains(err.Error(), "M3 ledger rollback refused") {
				t.Fatalf("unexpected Down failure: %v", err)
			}
			must(t, tx.Rollback(ctx))
			var count int
			must(t, pool.QueryRow(ctx, "SELECT count(*) FROM portfolios WHERE portfolio_id=$1", f.portfolio).Scan(&count))
			if count != 1 {
				t.Fatal("failed Down modified parent")
			}
			if !sequenceOnly {
				must(t, pool.QueryRow(ctx, "SELECT count(*) FROM transactions WHERE portfolio_id=$1", f.portfolio).Scan(&count))
				if count != 3 {
					t.Fatal("failed Down lost ledger")
				}
			}
		})
	}
}
