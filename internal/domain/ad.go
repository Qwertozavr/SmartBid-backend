package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrAdNotFound = errors.New("ad not found")
var ErrInvalidAd = errors.New("invalid ad")

type AdStatus string

const (
	AdStatusCreated   AdStatus = "created"
	AdStatusPublished AdStatus = "published"
	AdStatusCanceled  AdStatus = "canceled"
	AdStatusBought    AdStatus = "bought"
	AdStatusRemoved   AdStatus = "removed"
)

type Ad struct {
	ID          string
	Title       string
	Description string
	Price       int64
	Status      AdStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateAdInput struct {
	Title       string
	Description string
	Price       int64
	Status      AdStatus
}

func (input CreateAdInput) Validate() error {
	if strings.TrimSpace(input.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidAd)
	}
	if strings.TrimSpace(input.Description) == "" {
		return fmt.Errorf("%w: description is required", ErrInvalidAd)
	}
	if input.Price < 0 {
		return fmt.Errorf("%w: price must be greater than or equal to zero", ErrInvalidAd)
	}
	return nil
}
