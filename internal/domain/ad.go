package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrAdNotFound = errors.New("ad not found")
var ErrInvalidAd = errors.New("invalid ad")
var ErrAdNotActive = errors.New("ad is not active")

const MaxAdPhotoBytes = 5 << 20

type AdStatus string

const (
	AdStatusCreated   AdStatus = "created"
	AdStatusPublished AdStatus = "published"
	AdStatusCanceled  AdStatus = "canceled"
	AdStatusBought    AdStatus = "bought"
	AdStatusRemoved   AdStatus = "removed"
	AdStatusExpired   AdStatus = "expired"
)

type Ad struct {
	ID           string
	Title        string
	ChatId       int
	MessageId    int
	Description  *string
	Photo        []byte
	Price        int64
	PretendentID *int
	Status       AdStatus
	PublishedAt  *time.Time
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateAdInput struct {
	Title       string
	ChatId      int
	MessageId   int
	Description *string
	Photo       []byte
	Price       int64
	Status      AdStatus
}

type IncreaseAdPriceInput struct {
	AdID         string
	PretendentID int
}

type PublishAdInput struct {
	AdID   string
	ChatId int
}

type PublishAdUpdate struct {
	AdID        string
	PublishedAt time.Time
	ExpiresAt   time.Time
}

type UpdateAdPriceInput struct {
	AdID         string
	Price        int64
	PretendentID int
	Now          time.Time
}

type AdPriceUpdate struct {
	AdID  string
	Price int64
}

func (status AdStatus) CanTransitionTo(next AdStatus) bool {
	switch status {
	case AdStatusCreated:
		return next == AdStatusPublished || next == AdStatusRemoved
	case AdStatusPublished:
		return next == AdStatusBought || next == AdStatusExpired || next == AdStatusRemoved
	default:
		return false
	}
}

func (input IncreaseAdPriceInput) Validate() error {
	if strings.TrimSpace(input.AdID) == "" {
		return fmt.Errorf("%w: ad id is required", ErrInvalidAd)
	}
	if input.PretendentID <= 0 {
		return fmt.Errorf("%w: pretendent_id must be greater than zero", ErrInvalidAd)
	}
	return nil
}

func (input PublishAdInput) Validate() error {
	if strings.TrimSpace(input.AdID) == "" {
		return fmt.Errorf("%w: ad id is required", ErrInvalidAd)
	}
	if input.ChatId <= 0 {
		return fmt.Errorf("%w: chat_id must be greater than zero", ErrInvalidAd)
	}
	return nil
}

func (input CreateAdInput) Validate() error {
	if strings.TrimSpace(input.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidAd)
	}
	if input.Price < 0 {
		return fmt.Errorf("%w: price must be greater than or equal to zero", ErrInvalidAd)
	}
	if len(input.Photo) > MaxAdPhotoBytes {
		return fmt.Errorf("%w: photo must be less than or equal to 5 MB", ErrInvalidAd)
	}
	return nil
}
