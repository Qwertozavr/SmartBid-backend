package dto

import (
	"time"

	"smartbid-backend/internal/domain"
)

type CreateAdRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (request CreateAdRequest) ToDomainInput() domain.CreateAdInput {
	return domain.CreateAdInput{
		Title:       request.Title,
		Description: request.Description,
	}
}

type AdResponse struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Price       int64           `json:"price"`
	Status      domain.AdStatus `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func NewAdResponse(ad domain.Ad) AdResponse {
	return AdResponse{
		ID:          ad.ID,
		Title:       ad.Title,
		Description: ad.Description,
		Price:       ad.Price,
		Status:      ad.Status,
		CreatedAt:   ad.CreatedAt,
		UpdatedAt:   ad.UpdatedAt,
	}
}
