package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
)

const portfolioActiveNameConstraint = "portfolios_owner_normalized_active_uidx"

func mapPortfolioPersistenceError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return application.NewPersistenceError(application.ErrPortfolioNotFound, err)
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "23505":
			if postgresError.ConstraintName == portfolioActiveNameConstraint {
				return application.NewPersistenceError(application.ErrPortfolioNameConflict, err)
			}
			return application.NewPersistenceError(application.ErrPersistenceConflict, err)
		case "23503", "23514", "23P01":
			return application.NewPersistenceError(application.ErrPersistenceConflict, err)
		case "40001", "40P01":
			return application.NewPersistenceError(application.ErrPersistenceRetryable, err)
		}
	}
	return application.NewPersistenceError(application.ErrPersistenceFailure, err)
}
