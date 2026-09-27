//go:build integration

package database

import (
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/audit"
	platformdb "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestCallerTransactionIncludesLedgerIdentityAuditAndOutbox(t *testing.T) {
	pool, _ := testPools(t, 5)
	f := seed(t, pool)
	for _, commit := range []bool{false, true} {
		tx, err := pool.Begin(ctx)
		must(t, err)
		defer tx.Rollback(ctx)
		q := sqlcgen.New(tx)
		key := uuid.NewString()
		must(t, q.LockTransactionIdempotency(ctx, sqlcgen.LockTransactionIdempotencyParams{PortfolioID: f.portfolio, CommandScope: "transaction.create.v1", IdempotencyKey: key}))
		n, err := q.AllocateOnePortfolioSequence(ctx, f.portfolio)
		must(t, err)
		record := command(f, "FEE", n)
		insert(t, q, record)
		_, err = q.InsertCompletedTransactionIdempotency(ctx, completed(record, key))
		must(t, err)
		auditID, eventID := uid(), uid()
		must(t, platformdb.NewPlatformAuditStore(tx).Append(ctx, audit.Record{EventID: auditID.Bytes, OccurredAt: fixtureTime, Action: audit.ActionTransactionCreateSuccess, Result: audit.ResultSuccess, Severity: audit.SeverityInfo, ActorUserID: &f.owner.Bytes, CorrelationID: key, PortfolioID: &f.portfolio.Bytes, TransactionID: &record.TransactionID.Bytes}))
		// Synthetic event tests the existing seam only, not a production publisher.
		must(t, platformdb.NewPostgresOutboxStore(tx).Append(ctx, outbox.Event{ID: eventID.Bytes, Type: "persistence.fixture", Version: 1, AggregateType: "fixture", AggregateID: f.portfolio.Bytes, PortfolioID: f.portfolio.Bytes, OccurredAt: fixtureTime, CorrelationID: key, NextAttemptAt: fixtureTime, Payload: outbox.Payload{SchemaVersion: 1, References: []outbox.Reference{{Role: "record", ID: record.TransactionID.Bytes}}}}))
		if commit {
			must(t, tx.Commit(ctx))
		} else {
			must(t, tx.Rollback(ctx))
		}
		want := 0
		if commit {
			want = 1
		}
		for _, probe := range []struct {
			sql string
			id  pgtype.UUID
		}{
			{"SELECT count(*) FROM transactions WHERE transaction_id=$1", record.TransactionID},
			{"SELECT count(*) FROM transaction_idempotency WHERE primary_transaction_id=$1", record.TransactionID},
			{"SELECT count(*) FROM audit_logs WHERE audit_event_id=$1", auditID},
			{"SELECT count(*) FROM platform_outbox_events WHERE event_id=$1", eventID},
		} {
			var count int
			must(t, pool.QueryRow(ctx, probe.sql, probe.id).Scan(&count))
			if count != want {
				t.Fatal("caller transaction lost atomicity")
			}
		}
	}
}

func TestCorrectionRejectsMismatchedGroupsAndIncompleteRoles(t *testing.T) {
	pool, _ := testPools(t, 5)
	f := seed(t, pool)
	other := seed(t, pool)
	q := sqlcgen.New(pool)
	original := command(f, "BUY", 1)
	insert(t, q, original)
	for _, role := range []string{"replacement", "reversal"} {
		tx, err := pool.Begin(ctx)
		must(t, err)
		p := command(f, "BUY", 2)
		p.OriginatingCorrectionID = uid()
		if role == "reversal" {
			p.Kind = "REVERSAL"
			p.ReversalOfTransactionID = original.TransactionID
		} else {
			p.CorrectionOfTransactionID = original.TransactionID
		}
		insert(t, sqlcgen.New(tx), p)
		if err = tx.Commit(ctx); err == nil {
			t.Fatalf("orphan %s committed", role)
		}
	}
	for _, change := range []func(*sqlcgen.InsertTransactionCorrectionParams){
		func(c *sqlcgen.InsertTransactionCorrectionParams) { c.PortfolioID = other.portfolio },
		func(c *sqlcgen.InsertTransactionCorrectionParams) { c.OriginalTransactionID = uid() },
		func(c *sqlcgen.InsertTransactionCorrectionParams) { c.ReversalTransactionID = c.OriginalTransactionID },
		func(c *sqlcgen.InsertTransactionCorrectionParams) {
			c.ReplacementTransactionID = c.OriginalTransactionID
		},
		func(c *sqlcgen.InsertTransactionCorrectionParams) {
			c.ReplacementTransactionID = c.ReversalTransactionID
		},
		func(c *sqlcgen.InsertTransactionCorrectionParams) { c.ReplacementTransactionID = uid() },
	} {
		tx, err := pool.Begin(ctx)
		must(t, err)
		c := sqlcgen.InsertTransactionCorrectionParams{CorrectionID: uid(), PortfolioID: f.portfolio, OriginalTransactionID: original.TransactionID, ReversalTransactionID: uid(), ReplacementTransactionID: uid(), CreatedByUserID: f.owner, CreatedAt: instant(fixtureTime)}
		r := original
		r.TransactionID = c.ReversalTransactionID
		r.Kind = "REVERSAL"
		r.PortfolioSequence = 2
		r.ReversalOfTransactionID = original.TransactionID
		r.OriginatingCorrectionID = c.CorrectionID
		insert(t, sqlcgen.New(tx), r)
		replacement := original
		replacement.TransactionID = c.ReplacementTransactionID
		replacement.PortfolioSequence = 3
		replacement.CorrectionOfTransactionID = original.TransactionID
		replacement.OriginatingCorrectionID = c.CorrectionID
		insert(t, sqlcgen.New(tx), replacement)
		change(&c)
		_, err = sqlcgen.New(tx).InsertTransactionCorrection(ctx, c)
		if err == nil {
			err = tx.Commit(ctx)
		} else {
			must(t, tx.Rollback(ctx))
		}
		if err == nil {
			t.Fatal("invalid correction group committed")
		}
	}
}
