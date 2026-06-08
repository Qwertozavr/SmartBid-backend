package openrouter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"smartbid-backend/internal/price"
)

func TestNewClientRejectsEmptyAPIKey(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Config{
		BaseURL: "https://openrouter.example/api/v1",
		Model:   "example/model",
	})
	if err == nil {
		t.Fatal("NewClient() error = nil, want error")
	}
}

func TestClientEstimateSendsTextRequestAndReturnsPrice(t *testing.T) {
	t.Parallel()

	var got requestBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/api/v1/chat/completions")
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer secret-key" {
			t.Errorf("Authorization = %q, want Bearer secret-key", auth)
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}

		writeJSON(t, w, map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `{"recommended_price":125000}`,
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/api/v1///")
	description := "Почти не использовался"
	gotEstimate, err := client.Estimate(context.Background(), price.EstimateInput{
		Title:       "Ноутбук",
		Description: &description,
	})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if gotEstimate.RecommendedRubles != 125000 {
		t.Fatalf("RecommendedRubles = %d, want 125000", gotEstimate.RecommendedRubles)
	}

	if got.Model != "example/model" {
		t.Errorf("model = %q, want example/model", got.Model)
	}
	if got.ResponseFormat.Type != "json_object" {
		t.Errorf("response_format.type = %q, want json_object", got.ResponseFormat.Type)
	}
	if got.Temperature != 0.2 {
		t.Errorf("temperature = %v, want 0.2", got.Temperature)
	}
	if got.MaxTokens != 128 {
		t.Errorf("max_tokens = %v, want 128", got.MaxTokens)
	}
	if got.Reasoning.Effort != "none" {
		t.Errorf("reasoning.effort = %q, want none", got.Reasoning.Effort)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("messages = %#v, want system and user messages", got.Messages)
	}
	if got.Messages[0].Role != "system" || len(got.Messages[0].Content) != 1 {
		t.Fatalf("system message = %#v, want one system text part", got.Messages[0])
	}
	systemText := got.Messages[0].Content[0].Text
	for _, want := range []string{"recommended_price", "положительн", "цел", "российск", "руб"} {
		if !strings.Contains(systemText, want) {
			t.Errorf("system instruction does not contain %q: %q", want, systemText)
		}
	}
	for _, untrusted := range []string{"Ноутбук", description} {
		if strings.Contains(systemText, untrusted) {
			t.Errorf("system instruction contains untrusted listing data %q", untrusted)
		}
	}

	if got.Messages[1].Role != "user" || len(got.Messages[1].Content) != 1 || got.Messages[1].Content[0].Type != "text" {
		t.Fatalf("user message = %#v, want text only", got.Messages[1])
	}
	userText := got.Messages[1].Content[0].Text
	for _, want := range []string{"<listing_title>", "Ноутбук", "</listing_title>", "<listing_description>", description, "</listing_description>"} {
		if !strings.Contains(userText, want) {
			t.Errorf("user listing data does not contain %q: %q", want, userText)
		}
	}
}

func TestClientEstimateSendsPhotoAsDataURL(t *testing.T) {
	t.Parallel()

	photo := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	var got requestBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		writeJSON(t, w, map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{"content": `{"recommended_price":1}`},
			}},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.Estimate(context.Background(), price.EstimateInput{
		Title: "Фотоаппарат",
		Photo: photo,
	}); err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}

	if len(got.Messages) != 2 || got.Messages[0].Role != "system" || got.Messages[1].Role != "user" {
		t.Fatalf("messages = %#v, want system and user messages", got.Messages)
	}
	content := got.Messages[1].Content
	if len(content) != 2 {
		t.Fatalf("content length = %d, want 2", len(content))
	}
	if content[0].Type != "text" {
		t.Errorf("first content type = %q, want text", content[0].Type)
	}
	if content[1].Type != "image_url" || content[1].ImageURL == nil {
		t.Fatalf("second content = %#v, want image_url", content[1])
	}
	wantURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(photo)
	if content[1].ImageURL.URL != wantURL {
		t.Errorf("image URL = %q, want %q", content[1].ImageURL.URL, wantURL)
	}
}

func TestClientEstimateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		response   string
	}{
		{name: "rate limited", statusCode: http.StatusTooManyRequests, response: `provider-secret-429`},
		{name: "server error", statusCode: http.StatusInternalServerError, response: `provider-secret-500`},
		{name: "other non-2xx", statusCode: http.StatusBadRequest, response: `provider-secret-400`},
		{name: "malformed outer JSON", statusCode: http.StatusOK, response: `{`},
		{name: "missing choices", statusCode: http.StatusOK, response: `{"choices":[]}`},
		{name: "malformed inner JSON", statusCode: http.StatusOK, response: `{"choices":[{"message":{"content":"not-json"}}]}`},
		{name: "missing price", statusCode: http.StatusOK, response: `{"choices":[{"message":{"content":"{}"}}]}`},
		{name: "zero price", statusCode: http.StatusOK, response: `{"choices":[{"message":{"content":"{\"recommended_price\":0}"}}]}`},
		{name: "negative price", statusCode: http.StatusOK, response: `{"choices":[{"message":{"content":"{\"recommended_price\":-10}"}}]}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = io.WriteString(w, tt.response)
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)
			input := price.EstimateInput{
				Title: "prompt-secret-title",
				Photo: []byte("photo-secret-data"),
			}
			_, err := client.Estimate(context.Background(), input)
			if err == nil {
				t.Fatal("Estimate() error = nil, want error")
			}
			for _, secret := range []string{"secret-key", input.Title, string(input.Photo), tt.response} {
				if secret != "" && strings.Contains(err.Error(), secret) {
					t.Errorf("error leaks %q: %v", secret, err)
				}
			}
		})
	}
}

func TestClientEstimateIncludesStructuredProviderErrorDetails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"code":429,"message":"Free model rate limit exceeded"}}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.Estimate(context.Background(), price.EstimateInput{Title: "title"})
	if err == nil {
		t.Fatal("Estimate() error = nil, want error")
	}
	for _, want := range []string{"429", "Free model rate limit exceeded", "retry after 60"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Estimate() error = %q, want %q", err, want)
		}
	}
}

func TestClientEstimateRetriesTemporaryProviderErrorOnce(t *testing.T) {
	t.Parallel()

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"message":"temporarily rate-limited"}}`)
			return
		}
		writeJSON(t, w, map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{"content": `{"recommended_price":100}`},
			}},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	estimate, err := client.Estimate(context.Background(), price.EstimateInput{Title: "title"})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if estimate.RecommendedRubles != 100 {
		t.Fatalf("RecommendedRubles = %d, want 100", estimate.RecommendedRubles)
	}
}

func TestClientEstimateRetriesEmptyModelContentOnce(t *testing.T) {
	t.Parallel()

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		content := any(nil)
		if requests == 2 {
			content = `{"recommended_price":100}`
		}
		writeJSON(t, w, map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{"content": content},
			}},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	estimate, err := client.Estimate(context.Background(), price.EstimateInput{Title: "title"})
	if err != nil {
		t.Fatalf("Estimate() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if estimate.RecommendedRubles != 100 {
		t.Fatalf("RecommendedRubles = %d, want 100", estimate.RecommendedRubles)
	}
}

func TestClientEstimateReturnsTransportErrorWithoutLeakingRequest(t *testing.T) {
	t.Parallel()

	secrets := []string{"secret-key", "prompt-secret-title", "photo-secret-data"}
	transportErr := errors.New("transport failed: " + strings.Join(secrets, " "))
	client := &Client{
		apiKey:  "secret-key",
		baseURL: "https://openrouter.example/api/v1",
		model:   "example/model",
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, transportErr
		})},
	}

	_, err := client.Estimate(context.Background(), price.EstimateInput{
		Title: "prompt-secret-title",
		Photo: []byte("photo-secret-data"),
	})
	if err == nil {
		t.Fatal("Estimate() error = nil, want error")
	}
	if errors.Is(err, transportErr) {
		t.Errorf("Estimate() error wraps unsafe transport error: %v", err)
	}
	for _, secret := range secrets {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("error leaks %q: %v", secret, err)
		}
	}
}

func TestClientEstimatePreservesContextError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		context   func() (context.Context, context.CancelFunc)
		wantError error
	}{
		{
			name: "canceled",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			wantError: context.Canceled,
		},
		{
			name: "deadline exceeded",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			},
			wantError: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := tt.context()
			cancel()
			client := &Client{
				apiKey:  "secret-key",
				baseURL: "https://openrouter.example/api/v1",
				model:   "example/model",
				httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return nil, errors.New("unsafe transport error")
				})},
			}

			_, err := client.Estimate(ctx, price.EstimateInput{Title: "title"})
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("Estimate() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestClientEstimateReturnsGenericErrorOnConfiguredTimeout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client, err := NewClient(Config{
		APIKey:  "secret-key",
		BaseURL: "https://openrouter.example/api/v1",
		Model:   "example/model",
		Timeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})

	_, err = client.Estimate(ctx, price.EstimateInput{Title: "title"})
	if err == nil {
		t.Fatal("Estimate() error = nil, want error")
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Estimate() error = %v, want non-context error", err)
	}
	if ctx.Err() != nil {
		t.Fatalf("caller context error = %v, want nil", ctx.Err())
	}
	if err.Error() != "send openrouter request" {
		t.Fatalf("Estimate() error = %q, want generic transport error", err)
	}
}

func TestClientEstimateRejectsOversizedSuccessfulResponse(t *testing.T) {
	t.Parallel()

	const providerSecret = "provider-secret"
	response := strings.Repeat(" ", 1<<20+1) +
		`{"choices":[{"message":{"content":"{\"recommended_price\":1}"}}]}`
	client := &Client{
		apiKey:  "secret-key",
		baseURL: "https://openrouter.example/api/v1",
		model:   "example/model",
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(response + providerSecret)),
			}, nil
		})},
	}

	_, err := client.Estimate(context.Background(), price.EstimateInput{Title: "title"})
	if err == nil {
		t.Fatal("Estimate() error = nil, want oversized response error")
	}
	if strings.Contains(err.Error(), providerSecret) {
		t.Errorf("error leaks provider response: %v", err)
	}
}

func TestClientEstimatePreservesContextErrorWhileReadingResponse(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	const readerSecret = "provider-reader-secret"
	client := &Client{
		apiKey:  "secret-key",
		baseURL: "https://openrouter.example/api/v1",
		model:   "example/model",
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: &cancelingReadCloser{
					cancel: cancel,
					err:    errors.New(readerSecret),
				},
			}, nil
		})},
	}

	_, err := client.Estimate(ctx, price.EstimateInput{Title: "title"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Estimate() error = %v, want context.Canceled", err)
	}
	if strings.Contains(err.Error(), readerSecret) {
		t.Errorf("error leaks response reader error: %v", err)
	}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := NewClient(Config{
		APIKey:  "secret-key",
		BaseURL: baseURL,
		Model:   "example/model",
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

type requestBody struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			ImageURL *struct {
				URL string `json:"url"`
			} `json:"image_url"`
		} `json:"content"`
	} `json:"messages"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	Reasoning   struct {
		Effort string `json:"effort"`
	} `json:"reasoning"`
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type cancelingReadCloser struct {
	cancel context.CancelFunc
	err    error
}

func (r *cancelingReadCloser) Read([]byte) (int, error) {
	r.cancel()
	return 0, r.err
}

func (*cancelingReadCloser) Close() error {
	return nil
}
