//go:build integration

package database

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database/sqlcgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func completed(p sqlcgen.InsertTransactionParams, key string) sqlcgen.InsertCompletedTransactionIdempotencyParams {
	return sqlcgen.InsertCompletedTransactionIdempotencyParams{PortfolioID: p.PortfolioID, CommandScope: "transaction.create.v1", IdempotencyKey: key, FingerprintDigest: make([]byte, 32), PrimaryTransactionID: p.TransactionID, CompletedAt: instant(fixtureTime)}
}
func TestIdempotencyIdentityRetentionAndCleanup(t *testing.T) {
	pool, _ := testPools(t, 5)
	f := seed(t, pool)
	other := seed(t, pool)
	q := sqlcgen.New(pool)
	original := command(f, "FEE", 1)
	insert(t, q, original)
	otherOriginal := command(other, "FEE", 1)
	insert(t, q, otherOriginal)
	key := uuid.NewString()
	p := completed(original, key)
	saved, err := q.InsertCompletedTransactionIdempotency(ctx, p)
	must(t, err)
	if saved.ExpiresAt.Time.Sub(saved.CreatedAt.Time) != 365*24*time.Hour {
		t.Fatal("retention differs from 365 days")
	}
	if _, err = q.InsertCompletedTransactionIdempotency(ctx, p); err == nil {
		t.Fatal("duplicate committed identity accepted")
	}
	_, err = q.InsertCompletedTransactionIdempotency(ctx, completed(otherOriginal, key))
	must(t, err)
	tx, err := pool.Begin(ctx)
	must(t, err)
	c, replacement := correction(t, sqlcgen.New(tx), f, original, 2)
	correct := completed(replacement, key)
	correct.CommandScope = "transaction.correct.v1"
	correct.TargetTransactionID = original.TransactionID
	correct.CorrectionID = c.CorrectionID
	_, err = sqlcgen.New(tx).InsertCompletedTransactionIdempotency(ctx, correct)
	must(t, err)
	must(t, tx.Commit(ctx))
	for _, key := range []string{"", "short", "!123456789012345", strings.Repeat("a", 129), "abc defghijklmnop", "abcdefghijklmnøp"} {
		p := completed(original, key)
		if _, err = q.InsertCompletedTransactionIdempotency(ctx, p); err == nil {
			t.Fatal("invalid key accepted")
		}
	}
	for _, key := range []string{strings.Repeat("a", 16), strings.Repeat("b", 128), "A1234567890._~-x" + "z"} {
		_, err = q.InsertCompletedTransactionIdempotency(ctx, completed(original, key))
		must(t, err)
	}
	for _, change := range []func(*sqlcgen.InsertCompletedTransactionIdempotencyParams){
		func(p *sqlcgen.InsertCompletedTransactionIdempotencyParams) { p.CommandScope = "transaction.create.v2" },
		func(p *sqlcgen.InsertCompletedTransactionIdempotencyParams) { p.FingerprintDigest = make([]byte, 31) },
		func(p *sqlcgen.InsertCompletedTransactionIdempotencyParams) { p.FingerprintDigest = nil },
		func(p *sqlcgen.InsertCompletedTransactionIdempotencyParams) { p.PrimaryTransactionID = pgtype.UUID{} },
		func(p *sqlcgen.InsertCompletedTransactionIdempotencyParams) { p.PrimaryTransactionID = uid() },
		func(p *sqlcgen.InsertCompletedTransactionIdempotencyParams) {
			p.PrimaryTransactionID = otherOriginal.TransactionID
		},
		func(p *sqlcgen.InsertCompletedTransactionIdempotencyParams) {
			p.TargetTransactionID = original.TransactionID
		},
		func(p *sqlcgen.InsertCompletedTransactionIdempotencyParams) {
			p.CommandScope = "transaction.correct.v1"
		},
	} {
		p := completed(original, uuid.NewString())
		change(&p)
		if _, err = q.InsertCompletedTransactionIdempotency(ctx, p); err == nil {
			t.Fatal("invalid completed identity accepted")
		}
	}
	lookup := sqlcgen.GetUnexpiredTransactionIdempotencyParams{PortfolioID: f.portfolio, CommandScope: p.CommandScope, IdempotencyKey: key, AsOf: instant(saved.ExpiresAt.Time.Add(-time.Microsecond))}
	got, err := q.GetUnexpiredTransactionIdempotency(ctx, lookup)
	must(t, err)
	if got.PrimaryTransactionID != original.TransactionID {
		t.Fatal("result identity lost")
	}
	lookup.AsOf = saved.ExpiresAt
	if _, err = q.GetUnexpiredTransactionIdempotency(ctx, lookup); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("expiry boundary not exclusive")
	}
	count, err := q.DeleteExpiredTransactionIdempotencyKey(ctx, sqlcgen.DeleteExpiredTransactionIdempotencyKeyParams{PortfolioID: f.portfolio, CommandScope: p.CommandScope, IdempotencyKey: key, AsOf: saved.ExpiresAt})
	must(t, err)
	if count != 1 {
		t.Fatal("exact expiry cleanup")
	}
	// Other scope/Portfolio with the same key remains present (at the prior time).
	lookup.AsOf = instant(fixtureTime)
	lookup.CommandScope = "transaction.correct.v1"
	_, err = q.GetUnexpiredTransactionIdempotency(ctx, lookup)
	must(t, err)
	lookup.CommandScope = "transaction.create.v1"
	lookup.PortfolioID = other.portfolio
	_, err = q.GetUnexpiredTransactionIdempotency(ctx, lookup)
	must(t, err)
	fresh := completed(original, uuid.NewString())
	fresh.CompletedAt = saved.ExpiresAt
	_, err = q.InsertCompletedTransactionIdempotency(ctx, fresh)
	must(t, err)
	count, err = q.DeleteExpiredTransactionIdempotencyBatch(ctx, sqlcgen.DeleteExpiredTransactionIdempotencyBatchParams{AsOf: saved.ExpiresAt, BatchSize: 2})
	must(t, err)
	if count != 2 {
		t.Fatal("cleanup not bounded")
	}
	lookup = sqlcgen.GetUnexpiredTransactionIdempotencyParams{PortfolioID: f.portfolio, CommandScope: fresh.CommandScope, IdempotencyKey: fresh.IdempotencyKey, AsOf: saved.ExpiresAt}
	_, err = q.GetUnexpiredTransactionIdempotency(ctx, lookup)
	must(t, err)
	// Cleanup cannot delete financial facts.
	_, err = q.GetPortfolioTransaction(ctx, sqlcgen.GetPortfolioTransactionParams{PortfolioID: f.portfolio, TransactionID: original.TransactionID})
	must(t, err)
}

func TestSequenceSerializationAcrossIndependentPools(t *testing.T) {
	pool, second := testPools(t, 5)
	f := seed(t, pool)
	tx1, err := pool.Begin(ctx)
	must(t, err)
	defer tx1.Rollback(ctx)
	seq, err := sqlcgen.New(tx1).AllocateOnePortfolioSequence(ctx, f.portfolio)
	must(t, err)
	if seq != 1 {
		t.Fatal("first sequence != 1")
	}
	tx2, err := second.Begin(ctx)
	must(t, err)
	defer tx2.Rollback(ctx)
	// Server-enforced timeout proves the second connection cannot pass the lock
	// while tx1 remains open. No scheduling sleep or process mutex is involved.
	_, err = tx2.Exec(ctx, "SET LOCAL lock_timeout = '150ms'")
	must(t, err)
	if _, err = sqlcgen.New(tx2).AllocateOnePortfolioSequence(ctx, f.portfolio); err == nil {
		t.Fatal("concurrent sequence writer did not block")
	}
	must(t, tx2.Rollback(ctx))
	must(t, tx1.Commit(ctx))
	tx2, err = second.Begin(ctx)
	must(t, err)
	pair, err := sqlcgen.New(tx2).AllocateCorrectionPortfolioSequences(ctx, f.portfolio)
	must(t, err)
	if pair.ReversalSequence != 2 || pair.ReplacementSequence != 3 {
		t.Fatal("correction reservation not consecutive")
	}
	must(t, tx2.Commit(ctx))
	tx1, err = pool.Begin(ctx)
	must(t, err)
	seq, err = sqlcgen.New(tx1).AllocateOnePortfolioSequence(ctx, f.portfolio)
	must(t, err)
	if seq != 4 {
		t.Fatal("nonmonotonic sequence")
	}
	insert(t, sqlcgen.New(tx1), command(f, "FEE", seq))
	must(t, tx1.Rollback(ctx))
	tx2, err = second.Begin(ctx)
	must(t, err)
	seq, err = sqlcgen.New(tx2).AllocateOnePortfolioSequence(ctx, f.portfolio)
	must(t, err)
	if seq != 4 {
		t.Fatal("rollback consumed sequence")
	}
	must(t, tx2.Commit(ctx))
	// Simultaneous committed allocation requests return distinct positions.
	results := make(chan int64, 8)
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		p := pool
		if i%2 == 1 {
			p = second
		}
		go func() {
			tx, err := p.Begin(ctx)
			if err != nil {
				failures <- err
				return
			}
			defer tx.Rollback(ctx)
			n, err := sqlcgen.New(tx).AllocateOnePortfolioSequence(ctx, f.portfolio)
			if err == nil {
				err = tx.Commit(ctx)
			}
			if err != nil {
				failures <- err
			} else {
				results <- n
			}
		}()
	}
	seen := map[int64]bool{}
	for i := 0; i < 8; i++ {
		select {
		case err := <-failures:
			t.Fatal(err)
		case n := <-results:
			if seen[n] || n < 5 || n > 12 {
				t.Fatal("duplicate or out of order allocation")
			}
			seen[n] = true
		case <-time.After(10 * time.Second):
			t.Fatal("allocation deadlock")
		}
	}
}

func TestIdempotencyLockSerializesFinancialIdentityAndRollback(t *testing.T) {
	pool, second := testPools(t, 5)
	f := seed(t, pool)
	key := uuid.NewString()
	lock := sqlcgen.LockTransactionIdempotencyParams{PortfolioID: f.portfolio, CommandScope: "transaction.create.v1", IdempotencyKey: key}
	tx1, err := pool.Begin(ctx)
	must(t, err)
	defer tx1.Rollback(ctx)
	must(t, sqlcgen.New(tx1).LockTransactionIdempotency(ctx, lock))
	tx2, err := second.Begin(ctx)
	must(t, err)
	defer tx2.Rollback(ctx)
	_, err = tx2.Exec(ctx, "SET LOCAL lock_timeout = '150ms'")
	must(t, err)
	if err = sqlcgen.New(tx2).LockTransactionIdempotency(ctx, lock); err == nil {
		t.Fatal("advisory lock did not serialize connections")
	}
	must(t, tx2.Rollback(ctx))
	// Different scope/Portfolio/key must not share an intentional lock.
	tx2, err = second.Begin(ctx)
	must(t, err)
	_, err = tx2.Exec(ctx, "SET LOCAL lock_timeout = '150ms'")
	must(t, err)
	different := lock
	different.CommandScope = "transaction.correct.v1"
	must(t, sqlcgen.New(tx2).LockTransactionIdempotency(ctx, different))
	different = lock
	different.PortfolioID = uid()
	must(t, sqlcgen.New(tx2).LockTransactionIdempotency(ctx, different))
	different = lock
	different.IdempotencyKey = uuid.NewString()
	must(t, sqlcgen.New(tx2).LockTransactionIdempotency(ctx, different))
	must(t, tx2.Commit(ctx))
	q := sqlcgen.New(tx1)
	seq, err := q.AllocateOnePortfolioSequence(ctx, f.portfolio)
	must(t, err)
	p := command(f, "FEE", seq)
	insert(t, q, p)
	_, err = q.InsertCompletedTransactionIdempotency(ctx, completed(p, key))
	must(t, err)
	must(t, tx1.Rollback(ctx))
	lookup := sqlcgen.GetUnexpiredTransactionIdempotencyParams{PortfolioID: f.portfolio, CommandScope: lock.CommandScope, IdempotencyKey: key, AsOf: instant(fixtureTime)}
	if _, err = sqlcgen.New(pool).GetUnexpiredTransactionIdempotency(ctx, lookup); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("rolled-back identity persisted")
	}
	// Competing persistence clients use the seam; exactly one sees no result and
	// commits a synthetic fact. This is not a runtime command implementation.
	results := make(chan pgtype.UUID, 2)
	failures := make(chan error, 2)
	for i := 0; i < 2; i++ {
		p := pool
		if i == 1 {
			p = second
		}
		go func() {
			c, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			tx, err := p.Begin(c)
			if err != nil {
				failures <- err
				return
			}
			defer tx.Rollback(ctx)
			q := sqlcgen.New(tx)
			if err = q.LockTransactionIdempotency(c, lock); err != nil {
				failures <- err
				return
			}
			found, err := q.GetUnexpiredTransactionIdempotency(c, lookup)
			if errors.Is(err, pgx.ErrNoRows) {
				n, e := q.AllocateOnePortfolioSequence(c, f.portfolio)
				if e != nil {
					failures <- e
					return
				}
				record := command(f, "FEE", n)
				if _, e = q.InsertTransaction(c, record); e != nil {
					failures <- e
					return
				}
				found, err = q.InsertCompletedTransactionIdempotency(c, completed(record, key))
			}
			if err == nil {
				err = tx.Commit(c)
			}
			if err != nil {
				failures <- err
			} else {
				results <- found.PrimaryTransactionID
			}
		}()
	}
	var first pgtype.UUID
	for i := 0; i < 2; i++ {
		select {
		case err := <-failures:
			t.Fatal(err)
		case id := <-results:
			if i == 0 {
				first = id
			} else if id != first {
				t.Fatal("same identity committed different facts")
			}
		case <-time.After(12 * time.Second):
			t.Fatal("idempotency deadlock")
		}
	}
	var count int
	must(t, pool.QueryRow(ctx, "SELECT count(*) FROM transactions WHERE portfolio_id=$1", f.portfolio).Scan(&count))
	if count != 1 {
		t.Fatal("duplicate financial mutation")
	}
}
