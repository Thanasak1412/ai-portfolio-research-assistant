package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
)

func mapAssetPersistenceError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return application.NewPersistenceError(application.ErrAssetNotFound, err)
	}
	return application.NewPersistenceError(application.ErrPersistenceFailure, err)
}
