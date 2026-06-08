package app

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"smartbid-backend/internal/config"
)

func TestNewValidatesOpenRouterBeforeConnectingToDatabase(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{
		DatabaseURL:      "://invalid-database-url",
		OpenRouterAPIKey: "",
	}

	application, err := New(cfg, logger)
	if application != nil {
		application.Close()
		t.Fatal("expected application creation to fail")
	}
	if err == nil {
		t.Fatal("expected application creation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "openrouter") {
		t.Fatalf("expected openrouter error, got %q", err)
	}
}
