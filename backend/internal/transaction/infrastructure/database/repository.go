package database

import (
	"context"
	"errors"
	"time"

	asset "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type repository struct{ queries *sqlcgen.Queries }

func persistenceError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return application.ErrPersistence
}
func (r *repository) LockKey(ctx context.Context, pid portfolio.PortfolioID, scope, key string) error {
	return persistenceError(r.queries.LockTransactionIdempotency(ctx, sqlcgen.LockTransactionIdempotencyParams{PortfolioID: dbID(pid.Bytes()), CommandScope: scope, IdempotencyKey: key}))
}
func (r *repository) Idempotency(ctx context.Context, pid portfolio.PortfolioID, scope, key string, now time.Time) (application.Idempotency, bool, error) {
	row, err := r.queries.GetUnexpiredTransactionIdempotency(ctx, sqlcgen.GetUnexpiredTransactionIdempotencyParams{PortfolioID: dbID(pid.Bytes()), CommandScope: scope, IdempotencyKey: key, AsOf: dbTime(now)})
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = r.queries.DeleteExpiredTransactionIdempotencyKey(ctx, sqlcgen.DeleteExpiredTransactionIdempotencyKeyParams{PortfolioID: dbID(pid.Bytes()), CommandScope: scope, IdempotencyKey: key, AsOf: dbTime(now)})
		return application.Idempotency{}, false, persistenceError(err)
	}
	if err != nil {
		return application.Idempotency{}, false, persistenceError(err)
	}
	if len(row.FingerprintDigest) != 32 {
		return application.Idempotency{}, false, application.ErrPersistence
	}
	var result application.Idempotency
	copy(result.Fingerprint[:], row.FingerprintDigest)
	result.PrimaryID, err = domain.NewTransactionID(row.PrimaryTransactionID.Bytes)
	if err != nil {
		return result, false, application.ErrPersistence
	}
	if row.CorrectionID.Valid {
		result.CorrectionID, err = domain.NewCorrectionID(row.CorrectionID.Bytes)
		if err != nil {
			return result, false, application.ErrPersistence
		}
	}
	return result, true, nil
}
func (r *repository) Complete(ctx context.Context, pid portfolio.PortfolioID, scope, key string, digest [32]byte, target domain.TransactionID, result application.Result, now time.Time) error {
	var correction pgtype.UUID
	if result.Correction != nil {
		correction = dbID(result.Correction.Relationship.ID().Bytes())
	}
	_, err := r.queries.InsertCompletedTransactionIdempotency(ctx, sqlcgen.InsertCompletedTransactionIdempotencyParams{PortfolioID: dbID(pid.Bytes()), CommandScope: scope, IdempotencyKey: key, FingerprintDigest: digest[:], TargetTransactionID: dbID(target.Bytes()), PrimaryTransactionID: dbID(result.Transaction.ID().Bytes()), CorrectionID: correction, CompletedAt: dbTime(now)})
	return persistenceError(err)
}
func (r *repository) Allocate(ctx context.Context, pid portfolio.PortfolioID, correction bool) (int64, int64, error) {
	if correction {
		row, err := r.queries.AllocateCorrectionPortfolioSequences(ctx, dbID(pid.Bytes()))
		return row.ReversalSequence, row.ReplacementSequence, persistenceError(err)
	}
	sequence, err := r.queries.AllocateOnePortfolioSequence(ctx, dbID(pid.Bytes()))
	return sequence, 0, persistenceError(err)
}
func (r *repository) Replay(ctx context.Context, pid portfolio.PortfolioID, id asset.AssetID) ([]domain.Transaction, error) {
	rows, err := r.queries.ListAssetLedgerForReplay(ctx, sqlcgen.ListAssetLedgerForReplayParams{PortfolioID: dbID(pid.Bytes()), AssetID: dbID(id.Bytes())})
	if err != nil {
		return nil, persistenceError(err)
	}
	return mapRecords(rows)
}
func (r *repository) Get(ctx context.Context, pid portfolio.PortfolioID, id domain.TransactionID) (domain.Transaction, error) {
	row, err := r.queries.GetPortfolioTransaction(ctx, sqlcgen.GetPortfolioTransactionParams{PortfolioID: dbID(pid.Bytes()), TransactionID: dbID(id.Bytes())})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Transaction{}, application.ErrTransactionNotFound
	}
	if err != nil {
		return domain.Transaction{}, persistenceError(err)
	}
	return mapRecord(row)
}
func (r *repository) List(ctx context.Context, pid portfolio.PortfolioID, input application.HistoryInput) ([]domain.Transaction, error) {
	p := sqlcgen.ListPortfolioTransactionsParams{PortfolioID: dbID(pid.Bytes()), IncludeReversals: true, PageLimit: int32(input.Limit)}
	if input.Kind != nil {
		p.Kind = dbText(string(*input.Kind), true)
	}
	if input.IncludeReversals != nil {
		p.IncludeReversals = *input.IncludeReversals
	}
	if input.From != nil {
		p.EffectiveAtFrom = dbTime(*input.From)
	}
	if input.To != nil {
		p.EffectiveAtTo = dbTime(*input.To)
	}
	if input.After != nil {
		p.CursorEffectiveAt = dbTime(input.After.EffectiveAt)
		p.CursorPortfolioSequence = pgtype.Int8{Int64: input.After.Sequence, Valid: true}
		p.CursorTransactionID = dbID(input.After.ID.Bytes())
	}
	rows, err := r.queries.ListPortfolioTransactions(ctx, p)
	if err != nil {
		return nil, persistenceError(err)
	}
	return mapRecords(rows)
}
func (r *repository) DirectCorrection(ctx context.Context, pid portfolio.PortfolioID, id domain.TransactionID) (*domain.Correction, error) {
	row, err := r.queries.GetDirectTransactionCorrection(ctx, sqlcgen.GetDirectTransactionCorrectionParams{PortfolioID: dbID(pid.Bytes()), OriginalTransactionID: dbID(id.Bytes())})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, persistenceError(err)
	}
	correction, err := mapCorrection(row)
	if err != nil {
		return nil, err
	}
	return &correction, nil
}
func (r *repository) Correction(ctx context.Context, pid portfolio.PortfolioID, id domain.CorrectionID) (domain.CorrectionFacts, error) {
	row, err := r.queries.GetPortfolioCorrection(ctx, sqlcgen.GetPortfolioCorrectionParams{PortfolioID: dbID(pid.Bytes()), CorrectionID: dbID(id.Bytes())})
	if err != nil {
		return domain.CorrectionFacts{}, persistenceError(err)
	}
	relationship, err := mapCorrection(row)
	if err != nil {
		return domain.CorrectionFacts{}, err
	}
	reversal, err := r.Get(ctx, pid, relationship.ReversalID())
	if err != nil {
		return domain.CorrectionFacts{}, err
	}
	replacement, err := r.Get(ctx, pid, relationship.ReplacementID())
	if err != nil {
		return domain.CorrectionFacts{}, err
	}
	return domain.CorrectionFacts{Reversal: reversal, Replacement: replacement, Relationship: relationship}, nil
}
func (r *repository) Insert(ctx context.Context, fact domain.Transaction) error {
	params, err := recordParams(fact)
	if err != nil {
		return err
	}
	_, err = r.queries.InsertTransaction(ctx, params)
	return persistenceError(err)
}
func (r *repository) InsertCorrection(ctx context.Context, c domain.Correction) error {
	_, err := r.queries.InsertTransactionCorrection(ctx, sqlcgen.InsertTransactionCorrectionParams{CorrectionID: dbID(c.ID().Bytes()), PortfolioID: dbID(c.PortfolioID().Bytes()), OriginalTransactionID: dbID(c.OriginalID().Bytes()), ReversalTransactionID: dbID(c.ReversalID().Bytes()), ReplacementTransactionID: dbID(c.ReplacementID().Bytes()), CreatedByUserID: dbID(c.Actor().Bytes()), CreatedAt: dbTime(c.CreatedAt())})
	return persistenceError(err)
}

var _ application.Repository = (*repository)(nil)
