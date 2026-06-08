package config

import (
	"testing"
	"time"
)

func TestLoadDefaultsAdFinishedTopic(t *testing.T) {
	t.Setenv("KAFKA_AD_FINISHED_TOPIC", "")

	if got := Load().KafkaAdFinishedTopic; got != "ad-finished" {
		t.Fatalf("unexpected topic: %q", got)
	}
}

func TestLoadDefaultsOpenRouterConfig(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("OPENROUTER_BASE_URL", "")
	t.Setenv("OPENROUTER_MODEL", "")
	t.Setenv("OPENROUTER_TIMEOUT", "")

	cfg := Load()

	if cfg.OpenRouterAPIKey != "" {
		t.Fatalf("unexpected API key: %q", cfg.OpenRouterAPIKey)
	}
	if cfg.OpenRouterBaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("unexpected base URL: %q", cfg.OpenRouterBaseURL)
	}
	if cfg.OpenRouterModel != "nvidia/nemotron-nano-12b-v2-vl:free" {
		t.Fatalf("unexpected model: %q", cfg.OpenRouterModel)
	}
	if cfg.OpenRouterTimeout != 45*time.Second {
		t.Fatalf("unexpected timeout: %s", cfg.OpenRouterTimeout)
	}
}

func TestLoadCustomOpenRouterConfig(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "secret")
	t.Setenv("OPENROUTER_BASE_URL", "https://openrouter.example/api/v1")
	t.Setenv("OPENROUTER_MODEL", "example/model")
	t.Setenv("OPENROUTER_TIMEOUT", "27s")

	cfg := Load()

	if cfg.OpenRouterAPIKey != "secret" {
		t.Fatalf("unexpected API key: %q", cfg.OpenRouterAPIKey)
	}
	if cfg.OpenRouterBaseURL != "https://openrouter.example/api/v1" {
		t.Fatalf("unexpected base URL: %q", cfg.OpenRouterBaseURL)
	}
	if cfg.OpenRouterModel != "example/model" {
		t.Fatalf("unexpected model: %q", cfg.OpenRouterModel)
	}
	if cfg.OpenRouterTimeout != 27*time.Second {
		t.Fatalf("unexpected timeout: %s", cfg.OpenRouterTimeout)
	}
}

func TestLoadDefaultsInvalidOpenRouterTimeout(t *testing.T) {
	for _, value := range []string{"invalid", "0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("OPENROUTER_TIMEOUT", value)

			if got := Load().OpenRouterTimeout; got != 45*time.Second {
				t.Fatalf("unexpected timeout for %q: %s", value, got)
			}
		})
	}
}

func TestCreateAdTimeout(t *testing.T) {
	tests := []struct {
		name            string
		providerTimeout time.Duration
		want            time.Duration
	}{
		{
			name:            "uses minimum",
			providerTimeout: 15 * time.Second,
			want:            25 * time.Second,
		},
		{
			name:            "adds persistence margin",
			providerTimeout: 30 * time.Second,
			want:            40 * time.Second,
		},
		{
			name:            "saturates without overflow",
			providerTimeout: time.Duration(1<<63 - 1),
			want:            time.Duration(1<<63 - 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateAdTimeout(tt.providerTimeout); got != tt.want {
				t.Fatalf("CreateAdTimeout(%s) = %s, want %s", tt.providerTimeout, got, tt.want)
			}
		})
	}
}

func TestServerWriteTimeout(t *testing.T) {
	tests := []struct {
		name            string
		providerTimeout time.Duration
		want            time.Duration
	}{
		{
			name:            "adds response margin to minimum",
			providerTimeout: 15 * time.Second,
			want:            30 * time.Second,
		},
		{
			name:            "adds response margin to custom timeout",
			providerTimeout: 30 * time.Second,
			want:            45 * time.Second,
		},
		{
			name:            "saturates without overflow",
			providerTimeout: time.Duration(1<<63 - 1),
			want:            time.Duration(1<<63 - 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServerWriteTimeout(tt.providerTimeout); got != tt.want {
				t.Fatalf("ServerWriteTimeout(%s) = %s, want %s", tt.providerTimeout, got, tt.want)
			}
		})
	}
}
