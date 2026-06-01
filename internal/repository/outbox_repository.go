package repository

import (
	"context"

	"smartbid-backend/internal/domain"
)

type OutboxRepository interface {
	Create(ctx context.Context, event domain.OutboxEvent) error
	ClaimPending(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, cause error, maxAttempts int) error
}
