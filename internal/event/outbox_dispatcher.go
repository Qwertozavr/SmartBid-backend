package event

import (
	"context"
	"log/slog"
	"time"

	"smartbid-backend/internal/domain"
	"smartbid-backend/internal/repository"
)

type OutboxDispatcher struct {
	outbox      repository.OutboxRepository
	publisher   Publisher
	logger      *slog.Logger
	interval    time.Duration
	batchSize   int
	maxAttempts int
}

type OutboxDispatcherConfig struct {
	Interval    time.Duration
	BatchSize   int
	MaxAttempts int
}

func NewOutboxDispatcher(
	outbox repository.OutboxRepository,
	publisher Publisher,
	logger *slog.Logger,
	cfg OutboxDispatcherConfig,
) *OutboxDispatcher {
	if cfg.Interval == 0 {
		cfg.Interval = time.Second
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 10
	}
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 5
	}

	return &OutboxDispatcher{
		outbox:      outbox,
		publisher:   publisher,
		logger:      logger,
		interval:    cfg.Interval,
		batchSize:   cfg.BatchSize,
		maxAttempts: cfg.MaxAttempts,
	}
}

func (d *OutboxDispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		d.dispatch(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *OutboxDispatcher) dispatch(ctx context.Context) {
	events, err := d.outbox.ClaimPending(ctx, d.batchSize)
	if err != nil {
		if ctx.Err() == nil {
			d.logger.Error("failed to claim outbox events", "error", err)
		}
		return
	}

	for _, event := range events {
		if err := d.publisher.Publish(ctx, event); err != nil {
			d.handlePublishError(ctx, event, err)
			continue
		}

		if err := d.outbox.MarkPublished(ctx, event.ID); err != nil {
			d.logger.Error("failed to mark outbox event as published", "event_id", event.ID, "error", err)
		}
	}
}

func (d *OutboxDispatcher) handlePublishError(ctx context.Context, event domain.OutboxEvent, cause error) {
	if event.Attempts >= d.maxAttempts {
		if err := d.publisher.PublishDeadLetter(ctx, event, cause); err != nil {
			d.logger.Error("failed to publish outbox event to DLQ", "event_id", event.ID, "error", err)
		}
	}

	if err := d.outbox.MarkFailed(ctx, event.ID, cause, d.maxAttempts); err != nil {
		d.logger.Error("failed to mark outbox event as failed", "event_id", event.ID, "error", err)
		return
	}

	d.logger.Error("failed to publish outbox event", "event_id", event.ID, "error", cause)
}
