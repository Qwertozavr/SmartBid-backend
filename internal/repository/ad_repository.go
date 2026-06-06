package repository

import (
	"context"

	"smartbid-backend/internal/domain"
)

type AdRepository interface {
	Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error)
	FindByID(ctx context.Context, id string) (domain.Ad, error)
	FindByIDForUpdate(ctx context.Context, id string) (domain.Ad, error)
	Publish(ctx context.Context, input domain.PublishAdUpdate) error
	UpdatePrice(ctx context.Context, input domain.UpdateAdPriceInput) (domain.AdPriceUpdate, error)
	TransitionStatus(ctx context.Context, id string, from, to domain.AdStatus) error
}
