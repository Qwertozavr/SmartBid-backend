package service

import (
	"context"
	"encoding/json"
	"fmt"

	"smartbid-backend/internal/domain"
	"smartbid-backend/internal/repository"
)

type AdService struct {
	ads            repository.AdRepository
	outbox         repository.OutboxRepository
	transactor     repository.Transactor
	adCreatedTopic string
}

func NewAdService(
	ads repository.AdRepository,
	outbox repository.OutboxRepository,
	transactor repository.Transactor,
	adCreatedTopic string,
) *AdService {
	if adCreatedTopic == "" {
		adCreatedTopic = domain.AdCreatedTopic
	}

	return &AdService{
		ads:            ads,
		outbox:         outbox,
		transactor:     transactor,
		adCreatedTopic: adCreatedTopic,
	}
}

func (s *AdService) Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error) {
	if err := input.Validate(); err != nil {
		fmt.Println("ERROR Create Ad:", err)
		return domain.Ad{}, err
	}

	input.Price = 100

	input.Status = domain.AdStatusCreated

	var ad domain.Ad
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		createdAd, err := s.ads.Create(ctx, input)
		if err != nil {
			return err
		}

		if err := s.createAdCreatedOutboxEvent(ctx, createdAd.ID); err != nil {
			return err
		}

		ad = createdAd
		return nil
	})
	if err != nil {
		fmt.Println("ERROR Create Ad:", err)
		return domain.Ad{}, err
	}

	return ad, nil
}

func (s *AdService) FindByID(ctx context.Context, id string) (domain.Ad, error) {
	return s.ads.FindByID(ctx, id)
}

func (s *AdService) IncreasePrice(ctx context.Context, input domain.IncreaseAdPriceInput) (domain.AdPriceUpdate, error) {
	if err := input.Validate(); err != nil {
		return domain.AdPriceUpdate{}, err
	}

	ad, err := s.ads.FindByID(ctx, input.AdID)
	if err != nil {
		return domain.AdPriceUpdate{}, err
	}
	if ad.PretendentID != nil && *ad.PretendentID == input.PretendentID {
		return domain.AdPriceUpdate{}, fmt.Errorf("%w: pretendent_id must differ from previous pretendent_id", domain.ErrInvalidAd)
	}

	return s.ads.UpdatePrice(ctx, domain.UpdateAdPriceInput{
		AdID:         input.AdID,
		Price:        ad.Price * 105 / 100,
		PretendentID: input.PretendentID,
	})
}

func (s *AdService) createAdCreatedOutboxEvent(ctx context.Context, adID string) error {
	eventID, err := domain.NewEventID()
	if err != nil {
		return fmt.Errorf("create ad event id: %w", err)
	}

	payload, err := json.Marshal(domain.AdCreatedEvent{
		EventID: eventID,
		AdID:    adID,
	})
	if err != nil {
		return fmt.Errorf("marshal ad created event: %w", err)
	}

	err = s.outbox.Create(ctx, domain.OutboxEvent{
		ID:            eventID,
		Topic:         s.adCreatedTopic,
		EventType:     domain.AdCreatedEventType,
		AggregateType: "ad",
		AggregateID:   adID,
		Payload:       payload,
		Status:        domain.OutboxEventStatusPending,
	})
	if err != nil {
		return fmt.Errorf("create ad created outbox event: %w", err)
	}

	return nil
}
