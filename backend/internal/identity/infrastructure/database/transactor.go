package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/database/sqlcgen"
)

const transactionRollbackTimeout = 5 * time.Second

type PostgresTransactor struct{ pool *pgxpool.Pool }

func NewPostgresTransactor(pool *pgxpool.Pool) *PostgresTransactor {
	return &PostgresTransactor{pool: pool}
}

func (transactor *PostgresTransactor) WithinTransaction(
	ctx context.Context,
	operation func(context.Context, application.TransactionRepositories) error,
) error {
	if operation == nil {
		return application.ErrPersistenceFailure
	}
	tx, err := transactor.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return mapPersistenceError(err, application.ErrPersistenceFailure)
	}
	committed := false
	defer func() {
		if !committed {
			rollbackContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), transactionRollbackTimeout)
			defer cancel()
			_ = tx.Rollback(rollbackContext)
		}
	}()

	queries := sqlcgen.New(tx)
	repositories := transactionRepositories{
		users: newPostgresUserRepository(queries),
		sessions: &postgresRefreshSessionTransactionRepository{
			PostgresRefreshSessionRepository: newPostgresRefreshSessionRepository(queries),
		},
	}
	if err := operation(ctx, repositories); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return mapPersistenceError(err, application.ErrPersistenceFailure)
	}
	committed = true
	return nil
}

type transactionRepositories struct {
	users    application.UserRepository
	sessions application.RefreshSessionTransactionRepository
}

func (repositories transactionRepositories) Users() application.UserRepository {
	return repositories.users
}

func (repositories transactionRepositories) RefreshSessions() application.RefreshSessionTransactionRepository {
	return repositories.sessions
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

var _ application.Transactor = (*PostgresTransactor)(nil)
