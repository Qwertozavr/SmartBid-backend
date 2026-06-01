package dto

import (
	"time"

	"smartbid-backend/internal/domain"
)

type CreateAdRequest struct {
	Title       string  `json:"title"`
	ChatId      int     `json:"chat_id"`
	MessageId   int     `json:"message_id"`
	Description *string `json:"description,omitempty"`
	Photo       []byte  `json:"photo,omitempty"`
}

func (request CreateAdRequest) ToDomainInput() domain.CreateAdInput {
	return domain.CreateAdInput{
		Title:       request.Title,
		ChatId:      request.ChatId,
		MessageId:   request.MessageId,
		Description: request.Description,
		Photo:       request.Photo,
	}
}

type AdResponse struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	ChatId      int             `json:"chat_id"`
	MessageId   int             `json:"message_id"`
	Description *string         `json:"description"`
	Photo       []byte          `json:"photo,omitempty"`
	Price       int64           `json:"price"`
	Status      domain.AdStatus `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func NewAdResponse(ad domain.Ad) AdResponse {
	return AdResponse{
		ID:          ad.ID,
		Title:       ad.Title,
		ChatId:      ad.ChatId,
		MessageId:   ad.MessageId,
		Description: ad.Description,
		Photo:       ad.Photo,
		Price:       ad.Price,
		Status:      ad.Status,
		CreatedAt:   ad.CreatedAt,
		UpdatedAt:   ad.UpdatedAt,
	}
}
