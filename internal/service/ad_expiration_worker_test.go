package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

type fakeExpiredCompleter struct {
	calls chan int
}

func (f *fakeExpiredCompleter) CompleteExpired(_ context.Context, limit int) error {
	f.calls <- limit
	return nil
}

func TestAdExpirationWorkerRunsImmediatelyAndOnInterval(t *testing.T) {
	completer := &fakeExpiredCompleter{calls: make(chan int, 2)}
	worker := NewAdExpirationWorker(
		completer,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		AdExpirationWorkerConfig{Interval: time.Millisecond, BatchSize: 7},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Run(ctx)
	}()

	for range 2 {
		select {
		case limit := <-completer.calls:
			if limit != 7 {
				t.Fatalf("unexpected batch size: %d", limit)
			}
		case <-time.After(time.Second):
			t.Fatal("worker did not run")
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}

func TestNewAdExpirationWorkerUsesDefaults(t *testing.T) {
	worker := NewAdExpirationWorker(
		&fakeExpiredCompleter{calls: make(chan int, 1)},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		AdExpirationWorkerConfig{},
	)

	if worker.interval != time.Minute {
		t.Fatalf("interval = %v, want %v", worker.interval, time.Minute)
	}
	if worker.batchSize != 100 {
		t.Fatalf("batch size = %d, want 100", worker.batchSize)
	}
}
