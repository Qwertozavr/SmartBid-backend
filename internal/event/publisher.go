package event

import (
	"context"

	"smartbid-backend/internal/domain"
)

type Publisher interface {
	Publish(ctx context.Context, event domain.OutboxEvent) error
	PublishDeadLetter(ctx context.Context, event domain.OutboxEvent, cause error) error
	Close() error
}
