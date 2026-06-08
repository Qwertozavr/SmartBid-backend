package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"smartbid-backend/internal/domain"
	"smartbid-backend/internal/price"
	"smartbid-backend/internal/repository"
)

const (
	adLifetime                   = 24 * time.Hour
	fallbackAdPriceKopecks int64 = 100
	kopecksPerRuble        int64 = 100
)

type AdService struct {
	ads             repository.AdRepository
	priceEstimator  price.Estimator
	outbox          repository.OutboxRepository
	transactor      repository.Transactor
	adCreatedTopic  string
	adFinishedTopic string
	now             func() time.Time
}

func NewAdService(
	ads repository.AdRepository,
	priceEstimator price.Estimator,
	outbox repository.OutboxRepository,
	transactor repository.Transactor,
	adCreatedTopic string,
	adFinishedTopic string,
) *AdService {
	if adCreatedTopic == "" {
		adCreatedTopic = domain.AdCreatedTopic
	}
	if adFinishedTopic == "" {
		adFinishedTopic = domain.AdFinishedTopic
	}

	return &AdService{
		ads:             ads,
		priceEstimator:  priceEstimator,
		outbox:          outbox,
		transactor:      transactor,
		adCreatedTopic:  adCreatedTopic,
		adFinishedTopic: adFinishedTopic,
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

func (s *AdService) CompleteExpired(ctx context.Context, limit int) error {
	if limit <= 0 {
		return fmt.Errorf("%w: limit must be greater than zero", domain.ErrInvalidAd)
	}
	now := s.now().UTC()

	return s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		ads, err := s.ads.ClaimExpired(ctx, now, limit)
		if err != nil {
			return err
		}
		for _, ad := range ads {
			next := domain.AdStatusExpired
			if ad.PretendentID != nil {
				next = domain.AdStatusBought
			}
			if !ad.Status.CanTransitionTo(next) {
				return fmt.Errorf("%w: ad cannot transition from %s to %s", domain.ErrInvalidAd, ad.Status, next)
			}
			if err := s.ads.TransitionStatus(ctx, ad.ID, ad.Status, next); err != nil {
				return err
			}
			ad.Status = next
			if err := s.createAdFinishedOutboxEvent(ctx, ad); err != nil {
				return err
			}
		}
		return nil
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
		return domain.Ad{}, err
	}

	estimatedPrice, err := s.estimateAdPrice(ctx, input)
	if err != nil {
		return domain.Ad{}, err
	}
	input.Price = estimatedPrice
	input.Status = domain.AdStatusCreated

	ad, err := s.ads.Create(ctx, input)
	if err != nil {
		return domain.Ad{}, err
	}

	return ad, nil
}

func (s *AdService) estimateAdPrice(ctx context.Context, input domain.CreateAdInput) (int64, error) {
	if s.priceEstimator == nil {
		return fallbackAdPriceKopecks, nil
	}

	estimate, err := s.priceEstimator.Estimate(ctx, price.EstimateInput{
		Title:       input.Title,
		Description: input.Description,
		Photo:       input.Photo,
	})
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return 0, err
	}
	if err != nil || estimate.RecommendedRubles <= 0 || estimate.RecommendedRubles > math.MaxInt64/kopecksPerRuble {
		fmt.Println(err)
		return fallbackAdPriceKopecks, nil
	}

	return estimate.RecommendedRubles * kopecksPerRuble, nil
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

	increase := ad.Price / 20
	if increase > math.MaxInt64-ad.Price {
		return domain.AdPriceUpdate{}, fmt.Errorf("%w: increased price exceeds int64", domain.ErrInvalidAd)
	}

	return s.ads.UpdatePrice(ctx, domain.UpdateAdPriceInput{
		AdID:         input.AdID,
		Price:        ad.Price + increase,
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
