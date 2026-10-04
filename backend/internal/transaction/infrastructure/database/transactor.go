package database

import (
	"context"
	"time"

	asset "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	platform "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OwnershipBinder is supplied by Portfolio composition, which owns the query
// and shared lock. Transaction infrastructure never imports Portfolio SQL.
type OwnershipBinder func(pgx.Tx) portfolio.OwnershipReader
type AssetBinder func(pgx.Tx) asset.LookupReader
type PostgresTransactor struct {
	pool      *pgxpool.Pool
	ownership OwnershipBinder
	assets    AssetBinder
}

func NewPostgresTransactor(pool *pgxpool.Pool, ownership OwnershipBinder, assets AssetBinder) *PostgresTransactor {
	return &PostgresTransactor{pool, ownership, assets}
}

func (t *PostgresTransactor) WithinTransaction(ctx context.Context, operation func(context.Context, application.UnitOfWork) error) error {
	if t == nil || t.pool == nil || t.ownership == nil || t.assets == nil || operation == nil {
		return application.ErrPersistence
	}
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return persistenceError(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	u := application.UnitOfWork{Repository: &repository{sqlcgen.New(tx)}, Portfolios: t.ownership(tx), Assets: t.assets(tx), Audit: platform.NewPlatformAuditStore(tx), Outbox: platform.NewPostgresOutboxStore(tx)}
	if err := operation(ctx, u); err != nil {
		return err
	}
	return persistenceError(tx.Commit(ctx))
}
