package worker

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

type checkerStub struct{}

func (checkerStub) Ping(context.Context) error { return nil }

func TestRunStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		Run(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), checkerStub{}, time.Millisecond)
		close(done)
	}()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after cancellation")
	}
}
