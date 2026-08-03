package worker

import (
	"context"
	"log/slog"
	"time"
)

type DependencyChecker interface {
	Ping(context.Context) error
}

func Run(ctx context.Context, logger *slog.Logger, checker DependencyChecker, interval time.Duration) {
	logger.Info("worker started")
	defer logger.Info("worker stopped")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := checker.Ping(ctx); err != nil {
				logger.Warn("worker dependency check failed", "dependency", "database")
			} else {
				logger.Debug("worker dependency check passed")
			}
		}
	}
}
