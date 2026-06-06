package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smartbid-backend/internal/domain"
)

type fakeAdRepository struct {
	create       func(context.Context, domain.CreateAdInput) (domain.Ad, error)
	findByID     func(context.Context, string) (domain.Ad, error)
	publish      func(context.Context, domain.PublishAdUpdate) error
	updatePrice  func(context.Context, domain.UpdateAdPriceInput) (domain.AdPriceUpdate, error)
	updateStatus func(context.Context, string, domain.AdStatus) error
}

func (f *fakeAdRepository) Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error) {
	if f.create == nil {
		panic("unexpected Create call")
	}
	return f.create(ctx, input)
}

func (f *fakeAdRepository) FindByID(ctx context.Context, id string) (domain.Ad, error) {
	if f.findByID == nil {
		panic("unexpected FindByID call")
	}
	return f.findByID(ctx, id)
}

func (f *fakeAdRepository) Publish(ctx context.Context, input domain.PublishAdUpdate) error {
	if f.publish == nil {
		panic("unexpected Publish call")
	}
	return f.publish(ctx, input)
}

func (f *fakeAdRepository) UpdatePrice(ctx context.Context, input domain.UpdateAdPriceInput) (domain.AdPriceUpdate, error) {
	if f.updatePrice == nil {
		panic("unexpected UpdatePrice call")
	}
	return f.updatePrice(ctx, input)
}

func TestPublishSetsFixedTwentyFourHourTimer(t *testing.T) {
	now := time.Date(2026, time.June, 6, 12, 0, 0, 0, time.UTC)
	repository := &fakeAdRepository{
		findByID: func(_ context.Context, id string) (domain.Ad, error) {
			return domain.Ad{ID: id, ChatId: 12, Status: domain.AdStatusCreated}, nil
		},
		publish: func(_ context.Context, input domain.PublishAdUpdate) error {
			if !input.PublishedAt.Equal(now) {
				t.Fatalf("unexpected published_at: %v", input.PublishedAt)
			}
			if !input.ExpiresAt.Equal(now.Add(24 * time.Hour)) {
				t.Fatalf("unexpected expires_at: %v", input.ExpiresAt)
			}
			return nil
		},
	}
	service := NewAdService(repository, &fakeOutboxRepository{create: func(context.Context, domain.OutboxEvent) error {
		return nil
	}}, &fakeTransactor{}, "")
	service.now = func() time.Time { return now }

	if err := service.Publish(context.Background(), domain.PublishAdInput{AdID: "ad-1", ChatId: 12}); err != nil {
		t.Fatalf("publish ad: %v", err)
	}
}

func TestIncreasePriceRejectsExpiredAd(t *testing.T) {
	now := time.Date(2026, time.June, 7, 12, 0, 0, 0, time.UTC)
	repository := &fakeAdRepository{
		findByID: func(_ context.Context, id string) (domain.Ad, error) {
			return domain.Ad{ID: id, Status: domain.AdStatusPublished, ExpiresAt: &now}, nil
		},
	}
	service := NewAdService(repository, nil, nil, "")
	service.now = func() time.Time { return now }

	_, err := service.IncreasePrice(context.Background(), domain.IncreaseAdPriceInput{AdID: "ad-1", PretendentID: 42})
	if !errors.Is(err, domain.ErrAdNotActive) {
		t.Fatalf("expected inactive ad error, got %v", err)
	}
}

func (f *fakeAdRepository) UpdateStatus(ctx context.Context, id string, status domain.AdStatus) error {
	if f.updateStatus == nil {
		panic("unexpected UpdateStatus call")
	}
	return f.updateStatus(ctx, id, status)
}

type fakeOutboxRepository struct {
	create func(context.Context, domain.OutboxEvent) error
}

func (f *fakeOutboxRepository) Create(ctx context.Context, event domain.OutboxEvent) error {
	if f.create == nil {
		panic("unexpected Create call")
	}
	return f.create(ctx, event)
}

func (f *fakeOutboxRepository) ClaimPending(context.Context, int) ([]domain.OutboxEvent, error) {
	panic("unexpected ClaimPending call")
}

func (f *fakeOutboxRepository) MarkPublished(context.Context, string) error {
	panic("unexpected MarkPublished call")
}

func (f *fakeOutboxRepository) MarkFailed(context.Context, string, error, int) error {
	panic("unexpected MarkFailed call")
}

type fakeTransactor struct {
	withinTransaction func(context.Context, func(context.Context) error) error
}

func (f *fakeTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if f.withinTransaction != nil {
		return f.withinTransaction(ctx, fn)
	}
	return fn(ctx)
}

func TestCreateDoesNotCreateOutboxEvent(t *testing.T) {
	repository := &fakeAdRepository{
		create: func(_ context.Context, input domain.CreateAdInput) (domain.Ad, error) {
			return domain.Ad{ID: "ad-1", Title: input.Title}, nil
		},
	}
	service := NewAdService(repository, nil, nil, "")

	ad, err := service.Create(context.Background(), domain.CreateAdInput{Title: "title"})
	if err != nil {
		t.Fatalf("create ad: %v", err)
	}
	if ad.ID != "ad-1" {
		t.Fatalf("unexpected ad: %#v", ad)
	}
}

func TestPublishUpdatesAdStatusAndCreatesOutboxEvent(t *testing.T) {
	var statusUpdated bool
	repository := &fakeAdRepository{
		findByID: func(_ context.Context, id string) (domain.Ad, error) {
			return domain.Ad{ID: id, ChatId: 12, Status: domain.AdStatusCreated}, nil
		},
		publish: func(_ context.Context, input domain.PublishAdUpdate) error {
			if input.AdID != "ad-1" {
				t.Fatalf("unexpected publish update: %#v", input)
			}
			statusUpdated = true
			return nil
		},
	}
	outbox := &fakeOutboxRepository{
		create: func(_ context.Context, event domain.OutboxEvent) error {
			if !statusUpdated {
				t.Fatal("expected status to be updated before outbox event creation")
			}
			if event.Topic != domain.AdCreatedTopic || event.EventType != domain.AdCreatedEventType || event.AggregateID != "ad-1" {
				t.Fatalf("unexpected outbox event: %#v", event)
			}
			return nil
		},
	}
	transactor := &fakeTransactor{
		withinTransaction: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}
	service := NewAdService(repository, outbox, transactor, "")

	if err := service.Publish(context.Background(), domain.PublishAdInput{AdID: "ad-1", ChatId: 12}); err != nil {
		t.Fatalf("publish ad: %v", err)
	}
}

func TestPublishRejectsDifferentChatID(t *testing.T) {
	repository := &fakeAdRepository{
		findByID: func(_ context.Context, id string) (domain.Ad, error) {
			return domain.Ad{ID: id, ChatId: 12}, nil
		},
	}
	service := NewAdService(repository, nil, nil, "")

	err := service.Publish(context.Background(), domain.PublishAdInput{AdID: "ad-1", ChatId: 13})
	if !errors.Is(err, domain.ErrInvalidAd) {
		t.Fatalf("expected invalid ad error, got %v", err)
	}
}

func TestPublishDoesNotCreateOutboxEventWhenStatusUpdateFails(t *testing.T) {
	updateErr := errors.New("update status")
	repository := &fakeAdRepository{
		findByID: func(_ context.Context, id string) (domain.Ad, error) {
			return domain.Ad{ID: id, ChatId: 12, Status: domain.AdStatusCreated}, nil
		},
		publish: func(context.Context, domain.PublishAdUpdate) error {
			return updateErr
		},
	}
	service := NewAdService(repository, nil, &fakeTransactor{}, "")

	err := service.Publish(context.Background(), domain.PublishAdInput{AdID: "ad-1", ChatId: 12})
	if !errors.Is(err, updateErr) {
		t.Fatalf("expected update error, got %v", err)
	}
}
