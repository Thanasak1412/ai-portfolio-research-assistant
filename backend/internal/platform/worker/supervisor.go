package worker

import (
	"context"
	"log/slog"
	"time"
)

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
