//go:build integration

package database

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestTransactionFieldMatrixDecimalsAndSnapshots(t *testing.T) {
	pool, _ := testPools(t, 5)
	f := seed(t, pool)
	q := sqlcgen.New(pool)
	for i, kind := range []string{"BUY", "SELL", "DIVIDEND", "DEPOSIT", "WITHDRAWAL", "FEE"} {
		p := command(f, kind, int64(i+1))
		p.Note = txt("")
		p.ExternalReference = txt(" e\u0301 ")
		row := insert(t, q, p)
		if !row.Note.Valid || row.Note.String != "" || row.ExternalReference.String != p.ExternalReference.String {
			t.Fatal("text normalization or null/empty collapse")
		}
		p.TransactionID = uid()
		p.PortfolioSequence += 100
		p.Note = pgtype.Text{}
		p.ExternalReference = pgtype.Text{}
		row = insert(t, q, p)
		if row.Note.Valid || row.ExternalReference.Valid {
			t.Fatal("absent text changed")
		}
	}
	cases := []struct {
		name   string
		change func(*sqlcgen.InsertTransactionParams)
	}{
		{"adjustment", func(p *sqlcgen.InsertTransactionParams) { p.Kind = "ADJUSTMENT" }},
		{"unknown kind", func(p *sqlcgen.InsertTransactionParams) { p.Kind = "OTHER" }},
		{"currency", func(p *sqlcgen.InsertTransactionParams) { p.Currency = "EUR" }},
		{"missing quantity", func(p *sqlcgen.InsertTransactionParams) { p.Quantity = pgtype.Numeric{} }},
		{"missing unit price", func(p *sqlcgen.InsertTransactionParams) { p.UnitPrice = pgtype.Numeric{} }},
		{"missing fee", func(p *sqlcgen.InsertTransactionParams) { p.Fee = pgtype.Numeric{} }},
		{"trade amount", func(p *sqlcgen.InsertTransactionParams) { p.Amount = num("1") }},
		{"missing asset", func(p *sqlcgen.InsertTransactionParams) { p.AssetID = pgtype.UUID{} }},
		{"unknown asset", func(p *sqlcgen.InsertTransactionParams) { p.AssetID = uid() }},
		{"unknown portfolio", func(p *sqlcgen.InsertTransactionParams) { p.PortfolioID = uid() }},
		{"unknown actor", func(p *sqlcgen.InsertTransactionParams) { p.CreatedByUserID = uid() }},
		{"missing type snapshot", func(p *sqlcgen.InsertTransactionParams) { p.AssetTypeSnapshot = pgtype.Text{} }},
		{"missing exchange snapshot", func(p *sqlcgen.InsertTransactionParams) { p.AssetExchangeSnapshot = pgtype.Text{} }},
		{"missing currency snapshot", func(p *sqlcgen.InsertTransactionParams) { p.AssetCurrencySnapshot = pgtype.Text{} }},
		{"crypto", func(p *sqlcgen.InsertTransactionParams) { p.AssetTypeSnapshot = txt("CRYPTO") }},
		{"venue", func(p *sqlcgen.InsertTransactionParams) { p.AssetExchangeSnapshot = txt("LSE") }},
		{"snapshot currency", func(p *sqlcgen.InsertTransactionParams) { p.AssetCurrencySnapshot = txt("EUR") }},
		{"zero sequence", func(p *sqlcgen.InsertTransactionParams) { p.PortfolioSequence = 0 }},
		{"negative sequence", func(p *sqlcgen.InsertTransactionParams) { p.PortfolioSequence = -1 }},
		{"duplicate sequence", func(p *sqlcgen.InsertTransactionParams) { p.PortfolioSequence = 1 }},
		{"note bound", func(p *sqlcgen.InsertTransactionParams) { p.Note = txt(strings.Repeat("a", 2001)) }},
		{"reference bound", func(p *sqlcgen.InsertTransactionParams) { p.ExternalReference = txt(strings.Repeat("a", 257)) }},
		{"orphan correction link", func(p *sqlcgen.InsertTransactionParams) { p.OriginatingCorrectionID = uid() }},
		{"bare reversal", func(p *sqlcgen.InsertTransactionParams) { p.Kind = "REVERSAL" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := command(f, "BUY", 500)
			tc.change(&p)
			if _, err := q.InsertTransaction(ctx, p); err == nil {
				t.Fatal("invalid record accepted")
			}
		})
	}
	for _, field := range []string{"Quantity", "UnitPrice", "Fee", "Amount"} {
		for _, value := range []string{"-1", "NaN", "Infinity", "-Infinity", "0.0000000000001"} {
			t.Run(field+"_"+value, func(t *testing.T) {
				kind := "BUY"
				if field == "Amount" {
					kind = "FEE"
				}
				p := command(f, kind, 501)
				reflect.ValueOf(&p).Elem().FieldByName(field).Set(reflect.ValueOf(num(value)))
				if _, err := q.InsertTransaction(ctx, p); err == nil {
					t.Fatal("invalid decimal accepted")
				}
			})
		}
		if field != "Fee" {
			p := command(f, "BUY", 502)
			if field == "Amount" {
				p = command(f, "FEE", 502)
			}
			reflect.ValueOf(&p).Elem().FieldByName(field).Set(reflect.ValueOf(num("0")))
			if _, err := q.InsertTransaction(ctx, p); err == nil {
				t.Fatal("zero positive magnitude accepted")
			}
		}
	}
	p := command(f, "BUY", 503)
	p.Quantity = num("12345678901234567890123456.123456789012")
	p.UnitPrice = p.Quantity
	p.Fee = p.Quantity
	row := insert(t, q, p)
	if !reflect.DeepEqual(row.Quantity, p.Quantity) {
		t.Fatal("38 digit numeric lost precision")
	}
	p = command(f, "FEE", 504)
	p.Amount = num("12345678901234567890123456789012345678901234567890.123456789012")
	row = insert(t, q, p)
	if !reflect.DeepEqual(row.Amount, p.Amount) {
		t.Fatal("large exact numeric rounded")
	}
	// Every nontrade shape rejects forbidden financial dimensions.
	for _, kind := range []string{"DIVIDEND", "DEPOSIT", "WITHDRAWAL", "FEE"} {
		for _, field := range []string{"Quantity", "UnitPrice", "Fee"} {
			p := command(f, kind, 600)
			reflect.ValueOf(&p).Elem().FieldByName(field).Set(reflect.ValueOf(num("1")))
			if _, err := q.InsertTransaction(ctx, p); err == nil {
				t.Fatalf("%s accepted %s", kind, field)
			}
		}
		p := command(f, kind, 600)
		p.Amount = pgtype.Numeric{}
		if _, err := q.InsertTransaction(ctx, p); err == nil {
			t.Fatal("missing cash amount accepted")
		}
		p = command(f, kind, 600)
		if kind == "DIVIDEND" {
			p.AssetID = pgtype.UUID{}
			p.AssetTypeSnapshot = pgtype.Text{}
			p.AssetExchangeSnapshot = pgtype.Text{}
			p.AssetCurrencySnapshot = pgtype.Text{}
		} else {
			p.AssetID = f.asset
			p.AssetTypeSnapshot = txt("ETF")
			p.AssetExchangeSnapshot = txt("NYSEARCA")
			p.AssetCurrencySnapshot = txt("USD")
		}
		if _, err := q.InsertTransaction(ctx, p); err == nil {
			t.Fatal("invalid cash asset presence accepted")
		}
	}
	for i, exchange := range []string{"NYSE", "NASDAQ", "NYSEARCA", "AMEX"} {
		p := command(f, "BUY", int64(700+i))
		p.AssetTypeSnapshot = txt("ETF")
		p.AssetExchangeSnapshot = txt(exchange)
		insert(t, q, p)
	}
}

// Test helper only: wires predetermined immutable facts, not application replay
// or orchestration. The caller owns the transaction and chooses all identities.
func correction(t *testing.T, q *sqlcgen.Queries, f fixture, original sqlcgen.InsertTransactionParams, seq int64) (sqlcgen.TransactionCorrection, sqlcgen.InsertTransactionParams) {
	t.Helper()
	id := uid()
	r := original
	r.TransactionID = uid()
	r.Kind = "REVERSAL"
	r.PortfolioSequence = seq
	r.ReversalOfTransactionID = original.TransactionID
	r.CorrectionOfTransactionID = pgtype.UUID{}
	r.OriginatingCorrectionID = id
	r.Note = pgtype.Text{}
	r.ExternalReference = pgtype.Text{}
	replacement := original
	replacement.TransactionID = uid()
	replacement.PortfolioSequence = seq + 1
	replacement.CorrectionOfTransactionID = original.TransactionID
	replacement.ReversalOfTransactionID = pgtype.UUID{}
	replacement.OriginatingCorrectionID = id
	insert(t, q, r)
	insert(t, q, replacement)
	c, err := q.InsertTransactionCorrection(ctx, sqlcgen.InsertTransactionCorrectionParams{CorrectionID: id, PortfolioID: f.portfolio, OriginalTransactionID: original.TransactionID, ReversalTransactionID: r.TransactionID, ReplacementTransactionID: replacement.TransactionID, CreatedByUserID: f.owner, CreatedAt: instant(fixtureTime)})
	must(t, err)
	return c, replacement
}

func TestCorrectionAtomicityChainsAndScope(t *testing.T) {
	pool, _ := testPools(t, 5)
	f := seed(t, pool)
	other := seed(t, pool)
	q := sqlcgen.New(pool)
	for i, kind := range []string{"BUY", "DIVIDEND", "FEE"} {
		original := command(f, kind, int64(1+i*10))
		before := insert(t, q, original)
		tx, err := pool.Begin(ctx)
		must(t, err)
		defer tx.Rollback(ctx)
		first, replacement := correction(t, sqlcgen.New(tx), f, original, original.PortfolioSequence+1)
		must(t, tx.Commit(ctx))
		tx, err = pool.Begin(ctx)
		must(t, err)
		second, _ := correction(t, sqlcgen.New(tx), f, replacement, original.PortfolioSequence+3)
		must(t, tx.Commit(ctx))
		outgoing, err := q.GetDirectTransactionCorrection(ctx, sqlcgen.GetDirectTransactionCorrectionParams{PortfolioID: f.portfolio, OriginalTransactionID: replacement.TransactionID})
		must(t, err)
		if outgoing.CorrectionID != second.CorrectionID {
			t.Fatal("replacement cannot be corrected")
		}
		got, err := q.GetPortfolioTransaction(ctx, sqlcgen.GetPortfolioTransactionParams{PortfolioID: f.portfolio, TransactionID: original.TransactionID})
		must(t, err)
		if !reflect.DeepEqual(got, before) {
			t.Fatal("original changed")
		}
		got, err = q.GetPortfolioTransaction(ctx, sqlcgen.GetPortfolioTransactionParams{PortfolioID: f.portfolio, TransactionID: replacement.TransactionID})
		must(t, err)
		if got.OriginatingCorrectionID != first.CorrectionID {
			t.Fatal("chain erased incoming link")
		}
		if _, err = q.GetPortfolioCorrection(ctx, sqlcgen.GetPortfolioCorrectionParams{PortfolioID: other.portfolio, CorrectionID: first.CorrectionID}); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("cross-portfolio correction leak")
		}
		if _, err = q.GetPortfolioTransaction(ctx, sqlcgen.GetPortfolioTransactionParams{PortfolioID: other.portfolio, TransactionID: original.TransactionID}); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("cross-portfolio record leak")
		}
		r := original
		r.TransactionID = uid()
		r.PortfolioSequence = 100
		r.Kind = "REVERSAL"
		r.ReversalOfTransactionID = original.TransactionID
		r.OriginatingCorrectionID = uid()
		if _, err = q.InsertTransaction(ctx, r); err == nil {
			t.Fatal("second direct reversal accepted")
		}
		r.Kind = kind
		r.ReversalOfTransactionID = pgtype.UUID{}
		r.CorrectionOfTransactionID = original.TransactionID
		if _, err = q.InsertTransaction(ctx, r); err == nil {
			t.Fatal("second direct replacement accepted")
		}
	}
	original := command(f, "FEE", 200)
	before := insert(t, q, original)
	tx, err := pool.Begin(ctx)
	must(t, err)
	defer tx.Rollback(ctx)
	c, _ := correction(t, sqlcgen.New(tx), f, original, 201)
	must(t, tx.Rollback(ctx))
	if _, err = q.GetPortfolioCorrection(ctx, sqlcgen.GetPortfolioCorrectionParams{PortfolioID: f.portfolio, CorrectionID: c.CorrectionID}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("rolled-back correction persisted")
	}
	got, err := q.GetPortfolioTransaction(ctx, sqlcgen.GetPortfolioTransactionParams{PortfolioID: f.portfolio, TransactionID: original.TransactionID})
	must(t, err)
	if !reflect.DeepEqual(got, before) {
		t.Fatal("rollback modified original")
	}
	// A lone reversal can be inserted, but its missing group prevents COMMIT.
	tx, err = pool.Begin(ctx)
	must(t, err)
	r := original
	r.TransactionID = uid()
	r.PortfolioSequence = 201
	r.Kind = "REVERSAL"
	r.ReversalOfTransactionID = original.TransactionID
	r.OriginatingCorrectionID = uid()
	insert(t, sqlcgen.New(tx), r)
	if err = tx.Commit(ctx); err == nil {
		t.Fatal("partial correction committed")
	}
	r.PortfolioID = other.portfolio
	if _, err = q.InsertTransaction(ctx, r); err == nil {
		t.Fatal("cross-portfolio reversal accepted")
	}
	r.PortfolioID = f.portfolio
	r.TransactionID = r.ReversalOfTransactionID
	if _, err = q.InsertTransaction(ctx, r); err == nil {
		t.Fatal("self reversal accepted")
	}
}

func TestHistoryCursorFiltersAndReplayOrdering(t *testing.T) {
	pool, _ := testPools(t, 5)
	f := seed(t, pool)
	other := seed(t, pool)
	q := sqlcgen.New(pool)
	expected := []sqlcgen.Transaction{}
	for i := 1; i <= 5; i++ {
		p := command(f, "BUY", int64(i))
		if i < 3 {
			p.EffectiveAt = instant(fixtureTime.Add(-time.Hour))
		}
		expected = append(expected, insert(t, q, p))
	}
	insert(t, q, command(other, "BUY", 1))
	cash := command(f, "FEE", 6)
	insert(t, q, cash)
	p := sqlcgen.ListPortfolioTransactionsParams{PortfolioID: f.portfolio, IncludeReversals: true, PageLimit: 2}
	var ids []pgtype.UUID
	for {
		rows, err := q.ListPortfolioTransactions(ctx, p)
		must(t, err)
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			ids = append(ids, row.TransactionID)
		}
		last := rows[len(rows)-1]
		p.CursorEffectiveAt = last.EffectiveAt
		p.CursorPortfolioSequence = pgtype.Int8{Int64: last.PortfolioSequence, Valid: true}
		p.CursorTransactionID = last.TransactionID
	}
	if len(ids) != 6 || ids[0] != cash.TransactionID {
		t.Fatal("history pagination lost rows/order")
	}
	for i := 0; i < 5; i++ {
		if ids[i+1] != expected[4-i].TransactionID {
			t.Fatal("history tie ordering")
		}
	}
	p = sqlcgen.ListPortfolioTransactionsParams{PortfolioID: f.portfolio, IncludeReversals: true, PageLimit: 100, Kind: txt("BUY"), EffectiveAtFrom: instant(fixtureTime), EffectiveAtTo: instant(fixtureTime)}
	rows, err := q.ListPortfolioTransactions(ctx, p)
	must(t, err)
	if len(rows) != 3 {
		t.Fatal("inclusive range/kind filter")
	}
	p.EffectiveAtFrom = instant(fixtureTime.Add(time.Hour))
	p.EffectiveAtTo = instant(fixtureTime.Add(2 * time.Hour))
	rows, err = q.ListPortfolioTransactions(ctx, p)
	must(t, err)
	if len(rows) != 0 {
		t.Fatal("future read range")
	}
	replay, err := q.ListAssetLedgerForReplay(ctx, sqlcgen.ListAssetLedgerForReplayParams{PortfolioID: f.portfolio, AssetID: f.asset})
	must(t, err)
	if len(replay) != 5 {
		t.Fatal("replay scope")
	}
	for i, row := range replay {
		if row.TransactionID != expected[i].TransactionID {
			t.Fatal("replay ascending ordering")
		}
	}
	tx, err := pool.Begin(ctx)
	must(t, err)
	correction(t, sqlcgen.New(tx), f, cash, 7)
	must(t, tx.Commit(ctx))
	p = sqlcgen.ListPortfolioTransactionsParams{PortfolioID: f.portfolio, IncludeReversals: false, PageLimit: 100}
	rows, err = q.ListPortfolioTransactions(ctx, p)
	must(t, err)
	if len(rows) != 7 {
		t.Fatal("reversal visibility")
	}
	p.IncludeReversals = true
	p.Kind = txt("REVERSAL")
	rows, err = q.ListPortfolioTransactions(ctx, p)
	must(t, err)
	if len(rows) != 1 || rows[0].Kind != "REVERSAL" {
		t.Fatal("visible reversal filter")
	}
}
