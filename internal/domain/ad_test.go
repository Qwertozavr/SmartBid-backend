package domain

import "testing"

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
