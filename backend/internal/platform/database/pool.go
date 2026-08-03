package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/config"
)

func Open(ctx context.Context, applicationConfig config.Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(applicationConfig.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}
	poolConfig.MaxConns = applicationConfig.DatabaseMaxConnections
	poolConfig.MinConns = applicationConfig.DatabaseMinConnections

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return pool, nil
}

func Verify(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration) error {
	verificationContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := pool.Ping(verificationContext); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
