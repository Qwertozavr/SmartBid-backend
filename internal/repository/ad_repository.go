package repository

import (
	"context"

	"smartbid-backend/internal/domain"
)

type AdRepository interface {
	Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error)
	FindByID(ctx context.Context, id string) (domain.Ad, error)
}
