package worker

import (
	"context"
	"log/slog"
	"time"
)

// RunConfigured enforces ADR-023 at composition, before any claim cycle. A
// publisher must be supplied only by a separately approved receiver integration;
// configuration flags cannot substitute for that implementation.
func RunConfigured(ctx context.Context, logger *slog.Logger, checker DependencyChecker, heartbeat time.Duration, config DeliveryConfig, dependencies DeliveryDependencies) error {
	if logger == nil || checker == nil || heartbeat <= 0 {
		return ErrInvalidDeliveryDependencies
	}
	if err := config.Validate(); err != nil {
		return err
	}
	if dependencies.Publisher == nil {
		logger.Info("transaction publication inactive", "reason", "no_approved_receiver")
		Run(ctx, logger, checker, heartbeat)
		return ctx.Err()
	}
	delivery, err := NewDelivery(config, dependencies)
	if err != nil {
		return err
	}
	return RunWithDelivery(ctx, logger, checker, heartbeat, delivery)
}

// RunWithDelivery supervises the existing dependency heartbeat alongside an
// explicitly composed publisher. The composition root must supply a real
// receiver; nil does not silently disable delivery or acknowledge events.
func RunWithDelivery(ctx context.Context, logger *slog.Logger, checker DependencyChecker, heartbeat time.Duration, delivery *Delivery) error {
	if delivery == nil || logger == nil || checker == nil || heartbeat <= 0 {
		return ErrInvalidDeliveryDependencies
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	heartbeatDone := make(chan struct{})
	go func() { defer close(heartbeatDone); Run(ctx, logger, checker, heartbeat) }()
	err := delivery.Run(ctx)
	cancel()
	<-heartbeatDone
	return err
}
