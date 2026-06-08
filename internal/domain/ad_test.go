package domain

import (
	"errors"
	"testing"
)

func TestAdStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		from    AdStatus
		to      AdStatus
		allowed bool
	}{
		{AdStatusCreated, AdStatusPublished, true},
		{AdStatusCreated, AdStatusRemoved, true},
		{AdStatusPublished, AdStatusBought, true},
		{AdStatusPublished, AdStatusExpired, true},
		{AdStatusPublished, AdStatusRemoved, true},
		{AdStatusRemoved, AdStatusPublished, false},
		{AdStatusBought, AdStatusRemoved, false},
		{AdStatusExpired, AdStatusPublished, false},
	}

	for _, tt := range tests {
		if got := tt.from.CanTransitionTo(tt.to); got != tt.allowed {
			t.Fatalf("%s -> %s: expected %v, got %v", tt.from, tt.to, tt.allowed, got)
		}
	}
}

func TestAdInputsValidate(t *testing.T) {
	oversizedPhoto := make([]byte, MaxAdPhotoBytes+1)
	tests := []struct {
		name     string
		validate func() error
		wantErr  bool
	}{
		{name: "increase valid", validate: func() error {
			return (IncreaseAdPriceInput{AdID: "ad-1", PretendentID: 1}).Validate()
		}},
		{name: "increase missing ad id", validate: func() error {
			return (IncreaseAdPriceInput{PretendentID: 1}).Validate()
		}, wantErr: true},
		{name: "increase invalid pretendent", validate: func() error {
			return (IncreaseAdPriceInput{AdID: "ad-1"}).Validate()
		}, wantErr: true},
		{name: "publish valid", validate: func() error {
			return (PublishAdInput{AdID: "ad-1", ChatId: 1}).Validate()
		}},
		{name: "publish missing ad id", validate: func() error {
			return (PublishAdInput{ChatId: 1}).Validate()
		}, wantErr: true},
		{name: "publish invalid chat id", validate: func() error {
			return (PublishAdInput{AdID: "ad-1"}).Validate()
		}, wantErr: true},
		{name: "remove valid", validate: func() error {
			return (RemoveAdInput{AdID: "ad-1", ChatId: 1}).Validate()
		}},
		{name: "remove missing ad id", validate: func() error {
			return (RemoveAdInput{ChatId: 1}).Validate()
		}, wantErr: true},
		{name: "remove invalid chat id", validate: func() error {
			return (RemoveAdInput{AdID: "ad-1"}).Validate()
		}, wantErr: true},
		{name: "create valid", validate: func() error {
			return (CreateAdInput{Title: "title"}).Validate()
		}},
		{name: "create blank title", validate: func() error {
			return (CreateAdInput{Title: " "}).Validate()
		}, wantErr: true},
		{name: "create negative price", validate: func() error {
			return (CreateAdInput{Title: "title", Price: -1}).Validate()
		}, wantErr: true},
		{name: "create oversized photo", validate: func() error {
			return (CreateAdInput{Title: "title", Photo: oversizedPhoto}).Validate()
		}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validate()
			if tt.wantErr && !errors.Is(err, ErrInvalidAd) {
				t.Fatalf("expected invalid ad error, got %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestNewEventIDReturnsUUID(t *testing.T) {
	id, err := NewEventID()
	if err != nil {
		t.Fatalf("new event id: %v", err)
	}
	if len(id) != 36 || id[14] != '4' || (id[19] != '8' && id[19] != '9' && id[19] != 'a' && id[19] != 'b') {
		t.Fatalf("unexpected UUID v4: %q", id)
	}
}
