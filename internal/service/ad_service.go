package service

import (
	"context"

	"smartbid-backend/internal/domain"
	"smartbid-backend/internal/repository"
)

type AdService struct {
	ads repository.AdRepository
}

func NewAdService(ads repository.AdRepository) *AdService {
	return &AdService{ads: ads}
}

func (s *AdService) Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error) {
	if err := input.Validate(); err != nil {
		return domain.Ad{}, err
	}

	input.Price = 100

	input.Status = domain.AdStatusCreated

	return s.ads.Create(ctx, input)
}

func (s *AdService) FindByID(ctx context.Context, id string) (domain.Ad, error) {
	return s.ads.FindByID(ctx, id)
}
