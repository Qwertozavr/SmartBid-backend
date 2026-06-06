package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"smartbid-backend/internal/domain"
	"smartbid-backend/internal/repository"
)

const adLifetime = 24 * time.Hour

type AdService struct {
	ads             repository.AdRepository
	outbox          repository.OutboxRepository
	transactor      repository.Transactor
	adCreatedTopic  string
	adFinishedTopic string
	now             func() time.Time
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
		ads:             ads,
		outbox:          outbox,
		transactor:      transactor,
		adCreatedTopic:  adCreatedTopic,
		adFinishedTopic: domain.AdFinishedTopic,
		now:             time.Now,
	}
}

func (s *AdService) Remove(ctx context.Context, input domain.RemoveAdInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		ad, err := s.ads.FindByIDForUpdate(ctx, input.AdID)
		if err != nil {
			return err
		}
		if ad.ChatId != input.ChatId {
			return fmt.Errorf("%w: chat_id does not match ad chat_id", domain.ErrInvalidAd)
		}
		if !ad.Status.CanTransitionTo(domain.AdStatusRemoved) {
			return fmt.Errorf("%w: ad cannot be removed from status %s", domain.ErrInvalidAd, ad.Status)
		}
		if err := s.ads.TransitionStatus(ctx, ad.ID, ad.Status, domain.AdStatusRemoved); err != nil {
			return err
		}
		ad.Status = domain.AdStatusRemoved
		return s.createAdFinishedOutboxEvent(ctx, ad)
	})
}

func (s *AdService) createAdFinishedOutboxEvent(ctx context.Context, ad domain.Ad) error {
	eventID, err := domain.NewEventID()
	if err != nil {
		return fmt.Errorf("create ad finished event id: %w", err)
	}

	payload, err := json.Marshal(domain.AdFinishedEvent{
		EventID:      eventID,
		AdID:         ad.ID,
		Status:       ad.Status,
		PretendentID: ad.PretendentID,
		FinalPrice:   ad.Price,
	})
	if err != nil {
		return fmt.Errorf("marshal ad finished event: %w", err)
	}

	if err := s.outbox.Create(ctx, domain.OutboxEvent{
		ID:            eventID,
		Topic:         s.adFinishedTopic,
		EventType:     domain.AdFinishedEventType,
		AggregateType: "ad",
		AggregateID:   ad.ID,
		Payload:       payload,
		Status:        domain.OutboxEventStatusPending,
	}); err != nil {
		return fmt.Errorf("create ad finished outbox event: %w", err)
	}

	return nil
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
	now := s.now().UTC()
	if ad.Status != domain.AdStatusPublished || ad.ExpiresAt == nil || !ad.ExpiresAt.After(now) {
		return domain.AdPriceUpdate{}, domain.ErrAdNotActive
	}

	return s.ads.UpdatePrice(ctx, domain.UpdateAdPriceInput{
		AdID:         input.AdID,
		Price:        ad.Price * 105 / 100,
		PretendentID: input.PretendentID,
		Now:          now,
	})
}

func (s *AdService) Publish(ctx context.Context, input domain.PublishAdInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	ad, err := s.ads.FindByID(ctx, input.AdID)
	if err != nil {
		return err
	}
	if ad.ChatId != input.ChatId {
		return fmt.Errorf("%w: chat_id does not match ad chat_id", domain.ErrInvalidAd)
	}
	if !ad.Status.CanTransitionTo(domain.AdStatusPublished) {
		return fmt.Errorf("%w: ad cannot be published from status %s", domain.ErrInvalidAd, ad.Status)
	}

	publishedAt := s.now().UTC()

	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.ads.Publish(ctx, domain.PublishAdUpdate{
			AdID:        input.AdID,
			PublishedAt: publishedAt,
			ExpiresAt:   publishedAt.Add(adLifetime),
		}); err != nil {
			return err
		}

		return s.createAdCreatedOutboxEvent(ctx, input.AdID)
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
