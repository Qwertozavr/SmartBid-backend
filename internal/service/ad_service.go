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
