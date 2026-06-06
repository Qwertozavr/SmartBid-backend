package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smartbid-backend/internal/domain"
	"smartbid-backend/internal/http/handler"
)

type fakeAdService struct {
	create        func(context.Context, domain.CreateAdInput) (domain.Ad, error)
	findByID      func(context.Context, string) (domain.Ad, error)
	increasePrice func(context.Context, domain.IncreaseAdPriceInput) (domain.AdPriceUpdate, error)
	publish       func(context.Context, domain.PublishAdInput) error
	remove        func(context.Context, domain.RemoveAdInput) error
}

func (f *fakeAdService) Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error) {
	if f.create == nil {
		panic("unexpected Create call")
	}
	return f.create(ctx, input)
}

func (f *fakeAdService) FindByID(ctx context.Context, id string) (domain.Ad, error) {
	if f.findByID == nil {
		panic("unexpected FindByID call")
	}
	return f.findByID(ctx, id)
}

func (f *fakeAdService) IncreasePrice(ctx context.Context, input domain.IncreaseAdPriceInput) (domain.AdPriceUpdate, error) {
	if f.increasePrice == nil {
		panic("unexpected IncreasePrice call")
	}
	return f.increasePrice(ctx, input)
}

func (f *fakeAdService) Publish(ctx context.Context, input domain.PublishAdInput) error {
	if f.publish == nil {
		panic("unexpected Publish call")
	}
	return f.publish(ctx, input)
}

func (f *fakeAdService) Remove(ctx context.Context, input domain.RemoveAdInput) error {
	if f.remove == nil {
		panic("unexpected Remove call")
	}
	return f.remove(ctx, input)
}

func newTestRouter(ads handler.AdService) http.Handler {
	return New(Dependencies{
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		PingHandler: handler.NewPingHandler(),
		AdHandler:   handler.NewAdHandler(ads),
	})
}

func performRequest(t *testing.T, router http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

func decodeResponse[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()

	var payload T
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}

func TestPingReturnsOKAndSecurityHeaders(t *testing.T) {
	response := performRequest(t, newTestRouter(&fakeAdService{}), http.MethodGet, "/ping", nil)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected JSON content type, got %q", response.Header().Get("Content-Type"))
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected security headers to be applied")
	}

	payload := decodeResponse[map[string]string](t, response)
	if payload["status"] != "ok" {
		t.Fatalf("expected ok status, got %#v", payload)
	}
}

func TestCreateAdReturnsCreatedAd(t *testing.T) {
	createdAt := time.Date(2026, time.June, 6, 10, 0, 0, 0, time.UTC)
	expected := domain.Ad{
		ID:        "ad-1",
		Title:     "Laptop",
		ChatId:    12,
		MessageId: 34,
		Price:     100,
		Status:    domain.AdStatusCreated,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	service := &fakeAdService{
		create: func(_ context.Context, input domain.CreateAdInput) (domain.Ad, error) {
			if input.Title != expected.Title || input.ChatId != expected.ChatId || input.MessageId != expected.MessageId {
				t.Fatalf("unexpected create input: %#v", input)
			}
			return expected, nil
		},
	}

	response := performRequest(t, newTestRouter(service), http.MethodPost, "/api/v1/ads", []byte(
		`{"title":"Laptop","chat_id":12,"message_id":34}`,
	))

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	payload := decodeResponse[struct {
		ID      string          `json:"id"`
		Title   string          `json:"title"`
		Price   int64           `json:"price"`
		Status  domain.AdStatus `json:"status"`
		Message string          `json:"message"`
	}](t, response)
	if payload.ID != expected.ID || payload.Title != expected.Title || payload.Price != expected.Price || payload.Status != expected.Status || payload.Message != "Успешно" {
		t.Fatalf("unexpected response: %#v", payload)
	}
}

func TestCreateAdRejectsUnknownJSONField(t *testing.T) {
	response := performRequest(t, newTestRouter(&fakeAdService{}), http.MethodPost, "/api/v1/ads", []byte(
		`{"title":"Laptop","chat_id":12,"message_id":34,"unknown":true}`,
	))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	payload := decodeResponse[map[string]string](t, response)
	if payload["error"] != "invalid request body" {
		t.Fatalf("unexpected response: %#v", payload)
	}
}

func TestFindAdByIDReturnsNotFound(t *testing.T) {
	service := &fakeAdService{
		findByID: func(_ context.Context, id string) (domain.Ad, error) {
			if id != "missing-ad" {
				t.Fatalf("unexpected id: %q", id)
			}
			return domain.Ad{}, domain.ErrAdNotFound
		},
	}

	response := performRequest(t, newTestRouter(service), http.MethodGet, "/api/v1/ads/missing-ad", nil)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
	payload := decodeResponse[map[string]string](t, response)
	if payload["error"] != "ad not found" {
		t.Fatalf("unexpected response: %#v", payload)
	}
}

func TestFindAdByIDReturnsTimerFields(t *testing.T) {
	publishedAt := time.Date(2026, time.June, 6, 12, 0, 0, 0, time.UTC)
	expiresAt := publishedAt.Add(24 * time.Hour)
	service := &fakeAdService{
		findByID: func(_ context.Context, id string) (domain.Ad, error) {
			return domain.Ad{
				ID:          id,
				Title:       "Laptop",
				Status:      domain.AdStatusPublished,
				PublishedAt: &publishedAt,
				ExpiresAt:   &expiresAt,
			}, nil
		},
	}

	response := performRequest(t, newTestRouter(service), http.MethodGet, "/api/v1/ads/ad-1", nil)
	payload := decodeResponse[struct {
		PublishedAt *time.Time `json:"published_at"`
		ExpiresAt   *time.Time `json:"expires_at"`
	}](t, response)

	if payload.PublishedAt == nil || !payload.PublishedAt.Equal(publishedAt) {
		t.Fatalf("unexpected published_at: %v", payload.PublishedAt)
	}
	if payload.ExpiresAt == nil || !payload.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("unexpected expires_at: %v", payload.ExpiresAt)
	}
}

func TestIncreaseAdPriceReturnsUpdatedPrice(t *testing.T) {
	service := &fakeAdService{
		increasePrice: func(_ context.Context, input domain.IncreaseAdPriceInput) (domain.AdPriceUpdate, error) {
			if input.AdID != "ad-1" || input.PretendentID != 42 {
				t.Fatalf("unexpected increase input: %#v", input)
			}
			return domain.AdPriceUpdate{AdID: input.AdID, Price: 105}, nil
		},
	}

	response := performRequest(t, newTestRouter(service), http.MethodPost, "/api/v1/ads/ad-1/increase", []byte(
		`{"pretendent_id":42}`,
	))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	payload := decodeResponse[struct {
		ID      string `json:"id"`
		Price   int64  `json:"price"`
		Message string `json:"message"`
	}](t, response)
	if payload.ID != "ad-1" || payload.Price != 105 || payload.Message != "Успешно" {
		t.Fatalf("unexpected response: %#v", payload)
	}
}

func TestIncreaseAdPriceReturnsValidationError(t *testing.T) {
	service := &fakeAdService{
		increasePrice: func(_ context.Context, _ domain.IncreaseAdPriceInput) (domain.AdPriceUpdate, error) {
			return domain.AdPriceUpdate{}, errors.Join(domain.ErrInvalidAd, errors.New("pretendent_id must be greater than zero"))
		},
	}

	response := performRequest(t, newTestRouter(service), http.MethodPost, "/api/v1/ads/ad-1/increase", []byte(
		`{"pretendent_id":0}`,
	))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	payload := decodeResponse[map[string]string](t, response)
	if payload["error"] == "" {
		t.Fatalf("expected validation error, got %#v", payload)
	}
}

func TestPublishAdReturnsSuccessMessage(t *testing.T) {
	service := &fakeAdService{
		publish: func(_ context.Context, input domain.PublishAdInput) error {
			if input.AdID != "ad-1" || input.ChatId != 12 {
				t.Fatalf("unexpected publish input: %#v", input)
			}
			return nil
		},
	}

	response := performRequest(t, newTestRouter(service), http.MethodPost, "/api/v1/ads/ad-1/publish", []byte(
		`{"chat_id":12}`,
	))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	payload := decodeResponse[map[string]string](t, response)
	if payload["message"] != "Успешно" {
		t.Fatalf("unexpected response: %#v", payload)
	}
}

func TestPublishAdReturnsValidationError(t *testing.T) {
	service := &fakeAdService{
		publish: func(_ context.Context, _ domain.PublishAdInput) error {
			return errors.Join(domain.ErrInvalidAd, errors.New("chat_id does not match ad chat_id"))
		},
	}

	response := performRequest(t, newTestRouter(service), http.MethodPost, "/api/v1/ads/ad-1/publish", []byte(
		`{"chat_id":13}`,
	))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestRemoveAdReturnsSuccessMessage(t *testing.T) {
	service := &fakeAdService{
		remove: func(_ context.Context, input domain.RemoveAdInput) error {
			if input.AdID != "ad-1" || input.ChatId != 12 {
				t.Fatalf("unexpected remove input: %#v", input)
			}
			return nil
		},
	}

	response := performRequest(t, newTestRouter(service), http.MethodPost, "/api/v1/ads/ad-1/remove", []byte(
		`{"chat_id":12}`,
	))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	payload := decodeResponse[map[string]string](t, response)
	if payload["message"] != "Успешно" {
		t.Fatalf("unexpected response: %#v", payload)
	}
}
