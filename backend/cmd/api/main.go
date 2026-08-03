package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/config"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/logging"
)

type poolReadiness struct {
	ping func(context.Context) error
}

func (readiness poolReadiness) Ping(ctx context.Context) error { return readiness.ping(ctx) }

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

	rootContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pool, err := database.Open(rootContext, applicationConfig)
	if err != nil {
		logger.Error("database pool creation failed")
		os.Exit(1)
	}
	defer pool.Close()

	server := httpserver.New(logger, poolReadiness{ping: pool.Ping})
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.Listen(applicationConfig.HTTPAddress()) }()
	logger.Info("api started", "address", applicationConfig.HTTPAddress(), "environment", applicationConfig.Environment)

	select {
	case <-rootContext.Done():
	case serveErr := <-serveErrors:
		if serveErr != nil {
			logger.Error("api stopped unexpectedly", "error", serveErr)
		}
		stop()
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), applicationConfig.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("api graceful shutdown failed", "error", err)
	}
	logger.Info("api stopped", "shutdown_timeout", applicationConfig.ShutdownTimeout.String())
}
