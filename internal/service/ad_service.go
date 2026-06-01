package service

import (
	"context"
	"fmt"

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
		fmt.Println("ERROR Create Ad:", err)
		return domain.Ad{}, err
	}

	input.Price = 100

	input.Status = domain.AdStatusCreated

	ad, err := s.ads.Create(ctx, input)
	if err != nil {
		fmt.Println("ERROR Create Ad:", err)
		return domain.Ad{}, err
	}

	return ad, nil
}

func (s *AdService) FindByID(ctx context.Context, id string) (domain.Ad, error) {
	return s.ads.FindByID(ctx, id)
}
