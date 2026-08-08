package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

func mapPersistenceError(err error, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return application.NewPersistenceError(notFound, err)
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "23505":
			if postgresError.ConstraintName == "users_normalized_email_uidx" {
				return application.NewPersistenceError(application.ErrDuplicateIdentity, err)
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
