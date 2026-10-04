package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/config"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/logging"
	workerruntime "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/worker"
)

type poolChecker struct {
	ping func(context.Context) error
}

func (checker poolChecker) Ping(ctx context.Context) error { return checker.ping(ctx) }

func main() {
	applicationConfig, err := config.Load(os.LookupEnv)
	if err != nil {
		slog.Error("configuration validation failed", "error", err)
		os.Exit(1)
	}
	logger, err := logging.New(applicationConfig.Environment, applicationConfig.LogLevel)
	if err != nil {
		slog.Error("logging configuration failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pool, err := database.Open(ctx, applicationConfig)
	if err != nil {
		logger.Error("database pool creation failed")
		os.Exit(1)
	}
	defer pool.Close()

	// No concrete receiver is approved. ADR-023 keeps publication inactive;
	// deliberately do not construct an outbox store or a placeholder publisher.
	err = workerruntime.RunConfigured(ctx, logger, poolChecker{ping: pool.Ping}, applicationConfig.WorkerHeartbeatInterval, applicationConfig.OutboxDelivery, workerruntime.DeliveryDependencies{})
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("worker runtime failed")
		os.Exit(1)
	}
}
