package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"smartbid-backend/internal/domain"
	"smartbid-backend/internal/http/dto"
)

const maxRequestBodyBytes = 8 << 20

type AdService interface {
	Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error)
	FindByID(ctx context.Context, id string) (domain.Ad, error)
	IncreasePrice(ctx context.Context, input domain.IncreaseAdPriceInput) (domain.AdPriceUpdate, error)
	Publish(ctx context.Context, input domain.PublishAdInput) error
	Remove(ctx context.Context, input domain.RemoveAdInput) error
}

type AdHandler struct {
	ads           AdService
	createTimeout time.Duration
}

func NewAdHandler(ads AdService, createTimeout time.Duration) *AdHandler {
	return &AdHandler{
		ads:           ads,
		createTimeout: createTimeout,
	}
}

func (h *AdHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateAdRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.createTimeout)
	defer cancel()

	ad, err := h.ads.Create(ctx, request.ToDomainInput())
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAd) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create ad")
		return
	}

	writeJSON(w, http.StatusCreated, dto.NewCreateAdResponse(ad))
}

func (h *AdHandler) Remove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var request dto.RemoveAdRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.ads.Remove(ctx, request.ToDomainInput(id)); err != nil {
		if errors.Is(err, domain.ErrInvalidAd) || errors.Is(err, domain.ErrAdNotActive) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrAdNotFound) {
			writeError(w, http.StatusNotFound, "ad not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to remove ad")
		return
	}

	writeJSON(w, http.StatusOK, dto.NewSuccessResponse())
}

func (h *AdHandler) FindByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ad, err := h.ads.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrAdNotFound) {
			writeError(w, http.StatusNotFound, "ad not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to find ad")
		return
	}

	writeJSON(w, http.StatusOK, dto.NewAdResponse(ad))
}

func (h *AdHandler) IncreasePrice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var request dto.IncreaseAdPriceRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	priceUpdate, err := h.ads.IncreasePrice(ctx, request.ToDomainInput(id))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAd) || errors.Is(err, domain.ErrAdNotActive) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrAdNotFound) {
			writeError(w, http.StatusNotFound, "ad not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to increase ad price")
		return
	}

	writeJSON(w, http.StatusOK, dto.NewAdPriceUpdateResponse(priceUpdate))
}

func (h *AdHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var request dto.PublishAdRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err := h.ads.Publish(ctx, request.ToDomainInput(id))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAd) || errors.Is(err, domain.ErrAdNotActive) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrAdNotFound) {
			writeError(w, http.StatusNotFound, "ad not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to publish ad")
		return
	}

	writeJSON(w, http.StatusOK, dto.NewSuccessResponse())
}
