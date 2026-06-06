package service

import (
	"context"
	"log/slog"
	"time"
)

type ExpiredAdCompleter interface {
	CompleteExpired(ctx context.Context, limit int) error
}

type AdExpirationWorkerConfig struct {
	Interval  time.Duration
	BatchSize int
}

type AdExpirationWorker struct {
	completer ExpiredAdCompleter
	logger    *slog.Logger
	interval  time.Duration
	batchSize int
}

func NewAdExpirationWorker(
	completer ExpiredAdCompleter,
	logger *slog.Logger,
	cfg AdExpirationWorkerConfig,
) *AdExpirationWorker {
	if cfg.Interval == 0 {
		cfg.Interval = time.Minute
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	return &AdExpirationWorker{
		completer: completer,
		logger:    logger,
		interval:  cfg.Interval,
		batchSize: cfg.BatchSize,
	}
}

func (w *AdExpirationWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		if err := w.completer.CompleteExpired(ctx, w.batchSize); err != nil && ctx.Err() == nil {
			w.logger.Error("failed to complete expired ads", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
