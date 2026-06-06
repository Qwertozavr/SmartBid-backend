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

type IncreaseAdPriceRequest struct {
	PretendentID int `json:"pretendent_id"`
}

type PublishAdRequest struct {
	ChatId int `json:"chat_id"`
}

type RemoveAdRequest struct {
	ChatId int `json:"chat_id"`
}

const SuccessMessage = "Успешно"

type SuccessResponse struct {
	Message string `json:"message"`
}

type CreateAdResponse struct {
	AdResponse
	Message string `json:"message"`
}

type AdPriceUpdateResponse struct {
	ID      string `json:"id"`
	Price   int64  `json:"price"`
	Message string `json:"message"`
}

func (request IncreaseAdPriceRequest) ToDomainInput(adID string) domain.IncreaseAdPriceInput {
	return domain.IncreaseAdPriceInput{
		AdID:         adID,
		PretendentID: request.PretendentID,
	}
}

func (request PublishAdRequest) ToDomainInput(adID string) domain.PublishAdInput {
	return domain.PublishAdInput{
		AdID:   adID,
		ChatId: request.ChatId,
	}
}

func (request RemoveAdRequest) ToDomainInput(adID string) domain.RemoveAdInput {
	return domain.RemoveAdInput{
		AdID:   adID,
		ChatId: request.ChatId,
	}
}

func NewAdPriceUpdateResponse(update domain.AdPriceUpdate) AdPriceUpdateResponse {
	return AdPriceUpdateResponse{
		ID:      update.AdID,
		Price:   update.Price,
		Message: SuccessMessage,
	}
}

func NewSuccessResponse() SuccessResponse {
	return SuccessResponse{Message: SuccessMessage}
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
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	ChatId       int             `json:"chat_id"`
	MessageId    int             `json:"message_id"`
	Description  *string         `json:"description"`
	Photo        []byte          `json:"photo,omitempty"`
	Price        int64           `json:"price"`
	PretendentID *int            `json:"pretendent_id"`
	Status       domain.AdStatus `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func NewAdResponse(ad domain.Ad) AdResponse {
	return AdResponse{
		ID:           ad.ID,
		Title:        ad.Title,
		ChatId:       ad.ChatId,
		MessageId:    ad.MessageId,
		Description:  ad.Description,
		Photo:        ad.Photo,
		Price:        ad.Price,
		PretendentID: ad.PretendentID,
		Status:       ad.Status,
		CreatedAt:    ad.CreatedAt,
		UpdatedAt:    ad.UpdatedAt,
	}
}

func NewCreateAdResponse(ad domain.Ad) CreateAdResponse {
	return CreateAdResponse{
		AdResponse: NewAdResponse(ad),
		Message:    SuccessMessage,
	}
}
