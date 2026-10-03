//go:build integration

package database

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	assetcomposition "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/composition"
	asset "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/audit"
	platform "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
	portfoliocomposition "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/composition"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type applicationClock struct{}

func (applicationClock) Now() time.Time { return fixtureTime.Add(24 * time.Hour) }

type applicationIDs struct{}

func (applicationIDs) NewID() ([16]byte, error) { id, err := uuid.NewRandom(); return id, err }
func appService(t *testing.T, pool *pgxpool.Pool, decorate func(application.Transactor) application.Transactor) *application.Service {
	t.Helper()
	var tx application.Transactor = NewPostgresTransactor(pool, portfoliocomposition.BindOwnership, assetcomposition.BindLookup)
	if decorate != nil {
		tx = decorate(tx)
	}
	s, err := application.NewService(application.Dependencies{Transactor: tx, Clock: applicationClock{}, IDs: applicationIDs{}, Rejections: platform.NewPlatformAuditStore(pool)})
	must(t, err)
	return s
}
func appOwner(f fixture) (identity.Principal, portfolio.PortfolioID) {
	user, _ := identity.NewUserID(f.owner.Bytes)
	principal, _ := identity.NewPrincipal(user)
	pid, _ := portfolio.NewPortfolioID(f.portfolio.Bytes)
	return principal, pid
}
func appCommand(f fixture, kind domain.Kind) application.CommandInput {
	input := application.CommandInput{Kind: kind, Currency: domain.CurrencyUSD, EffectiveAt: fixtureTime}
	if kind == domain.KindBuy || kind == domain.KindSell || kind == domain.KindDividend {
		input.AssetID, _ = asset.NewAssetID(f.asset.Bytes)
	}
	if kind == domain.KindBuy || kind == domain.KindSell {
		input.Quantity, _ = domain.ParsePositiveDecimal("2")
		input.UnitPrice, _ = domain.ParsePositiveDecimal("10")
	} else {
		input.Amount, _ = domain.ParsePositiveDecimal("25")
	}
	return input
}
func appMeta() application.CommandMetadata {
	return application.CommandMetadata{IdempotencyKey: uuid.NewString(), CorrelationID: "m3-application-test"}
}
func counts(t *testing.T, pool *pgxpool.Pool) []int {
	t.Helper()
	result := []int{}
	for _, table := range []string{"transactions", "transaction_corrections", "transaction_idempotency", "audit_logs", "platform_outbox_events", "platform_outbox_streams", "transaction_portfolio_sequences"} {
		var count int
		must(t, pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count))
		result = append(result, count)
	}
	return result
}

func TestApplicationCreateReadReplayAndCorrection(t *testing.T) {
	pool, _ := testPools(t, 5)
	f := seed(t, pool)
	s := appService(t, pool, nil)
	principal, pid := appOwner(f)
	input, meta := appCommand(f, domain.KindBuy), appMeta()
	created, err := s.Create(ctx, principal, pid, input, meta)
	must(t, err)
	if created.Transaction.PortfolioSequence() != 1 {
		t.Fatal("sequence")
	}
	snapshot, ok := created.Transaction.Asset()
	if !ok || snapshot.ID() != input.AssetID || snapshot.Exchange() != "NASDAQ" {
		t.Fatal("canonical asset resolution")
	}
	fee, ok := created.Transaction.Fee()
	if !ok || fee.String() != "0" {
		t.Fatal("missing fee must be zero")
	}
	if !reflect.DeepEqual(counts(t, pool), []int{1, 0, 1, 1, 1, 1, 1}) {
		t.Fatal("non-atomic create counts", counts(t, pool))
	}
	retried, err := s.Create(ctx, principal, pid, input, meta)
	must(t, err)
	if retried.Transaction != created.Transaction {
		t.Fatal("idempotent result changed")
	}
	conflict := input
	conflict.Quantity, _ = domain.ParsePositiveDecimal("3")
	if _, err := s.Create(ctx, principal, pid, conflict, meta); !errors.Is(err, application.ErrIdempotencyConflict) {
		t.Fatalf("conflict: %v", err)
	}
	before, err := s.Get(ctx, principal, pid, created.Transaction.ID())
	must(t, err)
	correctMeta := appMeta()
	replacement := input
	replacement.Quantity, _ = domain.ParsePositiveDecimal("4")
	corrected, err := s.Correct(ctx, principal, pid, created.Transaction.ID(), replacement, correctMeta)
	must(t, err)
	if corrected.Correction == nil || corrected.Correction.Reversal.Kind() != domain.KindReversal || corrected.Transaction.PortfolioSequence() != 3 {
		t.Fatal("correction facts")
	}
	after, err := s.Get(ctx, principal, pid, created.Transaction.ID())
	must(t, err)
	if before.Transaction != after.Transaction || after.DirectCorrection == nil {
		t.Fatal("original changed or outgoing link missing")
	}
	for range 2 {
		retry, err := s.Correct(ctx, principal, pid, created.Transaction.ID(), replacement, correctMeta)
		must(t, err)
		if !reflect.DeepEqual(corrected, retry) {
			t.Fatal("correction replay changed")
		}
	}
	if _, err := s.Correct(ctx, principal, pid, created.Transaction.ID(), input, correctMeta); !errors.Is(err, application.ErrIdempotencyConflict) {
		t.Fatalf("correction conflict: %v", err)
	}
	if _, err := s.Correct(ctx, principal, pid, created.Transaction.ID(), input, appMeta()); !errors.Is(err, domain.ErrTransactionAlreadyCorrected) {
		t.Fatalf("already corrected: %v", err)
	}
	if _, err := s.Correct(ctx, principal, pid, corrected.Correction.Reversal.ID(), input, appMeta()); !errors.Is(err, domain.ErrTransactionNotCorrectable) {
		t.Fatalf("reversal correction: %v", err)
	}
	chained, err := s.Correct(ctx, principal, pid, corrected.Transaction.ID(), replacement, appMeta())
	must(t, err)
	chainRecord, err := s.Get(ctx, principal, pid, corrected.Transaction.ID())
	must(t, err)
	if chainRecord.Transaction.OriginatingCorrection().IsZero() || chainRecord.DirectCorrection == nil || chainRecord.DirectCorrection.ID() != chained.Correction.Relationship.ID() {
		t.Fatal("correction chain lost")
	}
	page, err := s.List(ctx, principal, pid, application.HistoryInput{Limit: 2})
	must(t, err)
	if len(page.Records) != 2 || page.Next == nil || page.Records[0].Transaction.PortfolioSequence() != 5 {
		t.Fatal("history order")
	}
	next, err := s.List(ctx, principal, pid, application.HistoryInput{Limit: 100, After: page.Next})
	must(t, err)
	if len(next.Records) != 3 || next.Next != nil {
		t.Fatal("cursor page")
	}
	future := fixtureTime.Add(365 * 24 * time.Hour)
	empty, err := s.List(ctx, principal, pid, application.HistoryInput{From: &future})
	must(t, err)
	if len(empty.Records) != 0 {
		t.Fatal("future read filters")
	}
	var emitted int
	must(t, pool.QueryRow(ctx, "SELECT count(*) FROM platform_outbox_events WHERE event_type='transaction.corrected.v1' AND jsonb_array_length(payload->'references')=4").Scan(&emitted))
	if emitted != 2 {
		t.Fatal("correction event references")
	}
}

func TestApplicationOwnershipEligibilityAndReplayRules(t *testing.T) {
	pool, _ := testPools(t, 5)
	f := seed(t, pool)
	other := seed(t, pool)
	s := appService(t, pool, nil)
	principal, pid := appOwner(f)
	outsider, _ := appOwner(other)
	input := appCommand(f, domain.KindBuy)
	if _, err := s.Create(ctx, outsider, pid, input, appMeta()); !errors.Is(err, application.ErrPortfolioNotFound) {
		t.Fatalf("ownership: %v", err)
	}
	if _, err := s.List(ctx, outsider, pid, application.HistoryInput{}); !errors.Is(err, application.ErrPortfolioNotFound) {
		t.Fatalf("list ownership: %v", err)
	}
	buy, err := s.Create(ctx, principal, pid, input, appMeta())
	must(t, err)
	if _, err := s.Get(ctx, outsider, pid, buy.Transaction.ID()); !errors.Is(err, application.ErrPortfolioNotFound) {
		t.Fatalf("get ownership: %v", err)
	}
	missing, _ := domain.NewTransactionID(uuid.New())
	if _, err := s.Get(ctx, principal, pid, missing); !errors.Is(err, application.ErrTransactionNotFound) {
		t.Fatal(err)
	}
	sell := appCommand(f, domain.KindSell)
	sell.Quantity, _ = domain.ParsePositiveDecimal("3")
	if _, err := s.Create(ctx, principal, pid, sell, appMeta()); !errors.Is(err, domain.ErrInsufficientOrderedQuantity) {
		t.Fatalf("oversell: %v", err)
	}
	sell.Quantity, _ = domain.ParsePositiveDecimal("2")
	sell.EffectiveAt = fixtureTime.Add(time.Hour)
	_, err = s.Create(ctx, principal, pid, sell, appMeta())
	must(t, err)
	replacement := input
	replacement.Quantity, _ = domain.ParsePositiveDecimal("1")
	if _, err := s.Correct(ctx, principal, pid, buy.Transaction.ID(), replacement, appMeta()); !errors.Is(err, application.ErrInvalidBackdatedLedger) {
		t.Fatalf("backdated validity: %v", err)
	}
	_, err = pool.Exec(ctx, "UPDATE assets SET asset_type='CRYPTO',exchange='CRYPTO' WHERE asset_id=$1", f.asset)
	must(t, err)
	if _, err := s.Create(ctx, principal, pid, input, appMeta()); !errors.Is(err, domain.ErrAssetFinanciallyIneligible) {
		t.Fatalf("crypto: %v", err)
	}
	_, err = pool.Exec(ctx, "UPDATE portfolios SET status='ARCHIVED',archived_at=$2,updated_at=$2 WHERE portfolio_id=$1", f.portfolio, fixtureTime.Add(time.Hour))
	must(t, err)
	if _, err := s.Create(ctx, principal, pid, appCommand(f, domain.KindDeposit), appMeta()); !errors.Is(err, application.ErrPortfolioArchived) {
		t.Fatalf("archived: %v", err)
	}
}

func TestApplicationConcurrentSameKeyAndPortfolioSequence(t *testing.T) {
	first, second := testPools(t, 5)
	f := seed(t, first)
	principal, pid := appOwner(f)
	services := []*application.Service{appService(t, first, nil), appService(t, second, nil)}
	for _, sameKey := range []bool{true, false} {
		meta := appMeta()
		const workers = 12
		results := make(chan application.Result, workers)
		errs := make(chan error, workers)
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				m := meta
				if !sameKey {
					m = appMeta()
				}
				result, err := services[i%2].Create(ctx, principal, pid, appCommand(f, domain.KindDeposit), m)
				results <- result
				errs <- err
			}(i)
		}
		close(start)
		wg.Wait()
		close(results)
		close(errs)
		for err := range errs {
			must(t, err)
		}
		seen := map[domain.TransactionID]bool{}
		sequences := map[int64]bool{}
		for result := range results {
			seen[result.Transaction.ID()] = true
			sequences[result.Transaction.PortfolioSequence()] = true
		}
		want := workers
		if sameKey {
			want = 1
		}
		if len(seen) != want || len(sequences) != want {
			t.Fatal("concurrent mutation duplicated or sequence lost")
		}
	}
	var count int
	must(t, first.QueryRow(ctx, "SELECT count(*) FROM transactions").Scan(&count))
	if count != 13 {
		t.Fatal("financial facts", count)
	}
}

type faultTransactor struct {
	application.Transactor
	stage string
}

func (f faultTransactor) WithinTransaction(ctx context.Context, operation func(context.Context, application.UnitOfWork) error) error {
	return f.Transactor.WithinTransaction(ctx, func(ctx context.Context, u application.UnitOfWork) error {
		u.Repository = faultRepository{u.Repository, f.stage}
		if f.stage == "audit" {
			u.Audit = faultAudit{u.Audit}
		}
		if f.stage == "outbox" {
			u.Outbox = faultOutbox{u.Outbox}
		}
		return operation(ctx, u)
	})
}

type faultRepository struct {
	application.Repository
	stage string
}

func (f faultRepository) Insert(ctx context.Context, fact domain.Transaction) error {
	if err := f.Repository.Insert(ctx, fact); err != nil {
		return err
	}
	if f.stage == "financial" || (f.stage == "replacement" && !fact.CorrectionOf().IsZero()) {
		return application.ErrPersistence
	}
	return nil
}
func (f faultRepository) InsertCorrection(ctx context.Context, c domain.Correction) error {
	if err := f.Repository.InsertCorrection(ctx, c); err != nil {
		return err
	}
	if f.stage == "relationship" {
		return application.ErrPersistence
	}
	return nil
}
func (f faultRepository) Complete(ctx context.Context, pid portfolio.PortfolioID, scope, key string, digest [32]byte, target domain.TransactionID, result application.Result, now time.Time) error {
	if err := f.Repository.Complete(ctx, pid, scope, key, digest, target, result, now); err != nil {
		return err
	}
	if f.stage == "idempotency" {
		return application.ErrPersistence
	}
	return nil
}

type faultAudit struct{ audit.Store }

func (f faultAudit) Append(ctx context.Context, r audit.Record) error {
	if err := f.Store.Append(ctx, r); err != nil {
		return err
	}
	return application.ErrPersistence
}

type faultOutbox struct{ outbox.Appender }

func (f faultOutbox) Append(ctx context.Context, e outbox.Event) error {
	if err := f.Appender.Append(ctx, e); err != nil {
		return err
	}
	return application.ErrPersistence
}

func TestApplicationRollsBackEveryWriteBoundary(t *testing.T) {
	for _, correction := range []bool{false, true} {
		for _, stage := range []string{"financial", "replacement", "audit", "outbox", "idempotency", "relationship"} {
			if !correction && (stage == "relationship" || stage == "replacement") {
				continue
			}
			t.Run(stage+map[bool]string{false: "/create", true: "/correct"}[correction], func(t *testing.T) {
				pool, _ := testPools(t, 5)
				f := seed(t, pool)
				principal, pid := appOwner(f)
				base := appService(t, pool, nil)
				var target domain.TransactionID
				if correction {
					result, err := base.Create(ctx, principal, pid, appCommand(f, domain.KindDeposit), appMeta())
					must(t, err)
					target = result.Transaction.ID()
				}
				before := counts(t, pool)
				failing := appService(t, pool, func(tx application.Transactor) application.Transactor { return faultTransactor{tx, stage} })
				var err error
				if correction {
					_, err = failing.Correct(ctx, principal, pid, target, appCommand(f, domain.KindDeposit), appMeta())
				} else {
					_, err = failing.Create(ctx, principal, pid, appCommand(f, domain.KindDeposit), appMeta())
				}
				if !errors.Is(err, application.ErrPersistence) {
					t.Fatalf("expected injected failure: %v", err)
				}
				if after := counts(t, pool); !reflect.DeepEqual(before, after) {
					t.Fatalf("partial state survived: before=%v after=%v", before, after)
				}
			})
		}
	}
}

func TestApplicationConcurrentCorrectionHasIndependentScope(t *testing.T) {
	first, second := testPools(t, 5)
	f := seed(t, first)
	principal, pid := appOwner(f)
	services := []*application.Service{appService(t, first, nil), appService(t, second, nil)}
	meta := appMeta()
	input := appCommand(f, domain.KindDeposit)
	original, err := services[0].Create(ctx, principal, pid, input, meta)
	must(t, err)
	const workers = 8
	results := make(chan application.Result, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result, err := services[i%2].Correct(ctx, principal, pid, original.Transaction.ID(), input, meta)
			results <- result
			errs <- err
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		must(t, err)
	}
	seen := map[domain.CorrectionID]bool{}
	for result := range results {
		if result.Correction == nil {
			t.Fatal("missing correction")
		}
		seen[result.Correction.Relationship.ID()] = true
	}
	if len(seen) != 1 {
		t.Fatal("duplicate concurrent correction")
	}
	got := counts(t, first)
	if got[0] != 3 || got[1] != 1 || got[2] != 2 || got[4] != 2 {
		t.Fatal("create/correct scopes or financial cardinality", got)
	}
}

type ownershipPause struct {
	application.PortfolioReader
	entered chan struct{}
	release chan struct{}
}

func (p ownershipPause) GetPortfolio(ctx context.Context, principal identity.Principal, id portfolio.PortfolioID) (portfolio.Portfolio, error) {
	value, err := p.PortfolioReader.GetPortfolio(ctx, principal, id)
	if err != nil {
		return value, err
	}
	close(p.entered)
	select {
	case <-p.release:
		return value, nil
	case <-ctx.Done():
		return portfolio.Portfolio{}, ctx.Err()
	}
}

type decoratedTransaction struct {
	application.Transactor
	decorate func(application.UnitOfWork) application.UnitOfWork
}

func (d decoratedTransaction) WithinTransaction(ctx context.Context, operation func(context.Context, application.UnitOfWork) error) error {
	return d.Transactor.WithinTransaction(ctx, func(ctx context.Context, u application.UnitOfWork) error { return operation(ctx, d.decorate(u)) })
}

func TestApplicationOwnershipLockExcludesConcurrentArchive(t *testing.T) {
	first, second := testPools(t, 5)
	f := seed(t, first)
	principal, pid := appOwner(f)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	s := appService(t, first, func(tx application.Transactor) application.Transactor {
		return decoratedTransaction{tx, func(u application.UnitOfWork) application.UnitOfWork {
			u.Portfolios = ownershipPause{u.Portfolios, entered, release}
			return u
		}}
	})
	done := make(chan error, 1)
	go func() {
		_, err := s.Create(ctx, principal, pid, appCommand(f, domain.KindDeposit), appMeta())
		done <- err
	}()
	<-entered
	tx, err := second.Begin(ctx)
	must(t, err)
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "SET LOCAL lock_timeout='10ms'")
	must(t, err)
	_, err = tx.Exec(ctx, "UPDATE portfolios SET status='ARCHIVED', archived_at=$2, updated_at=$2 WHERE portfolio_id=$1", f.portfolio, fixtureTime.Add(time.Hour))
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "55P03" {
		t.Fatalf("archive must wait for financial transaction: %v", err)
	}
	must(t, tx.Rollback(ctx))
	releaseOnce.Do(func() { close(release) })
	must(t, <-done)
	_, err = second.Exec(ctx, "UPDATE portfolios SET status='ARCHIVED', archived_at=$2, updated_at=$2 WHERE portfolio_id=$1", f.portfolio, fixtureTime.Add(time.Hour))
	must(t, err)
}
