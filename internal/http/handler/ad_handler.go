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

const maxRequestBodyBytes = 1 << 20

type AdService interface {
	Create(ctx context.Context, input domain.CreateAdInput) (domain.Ad, error)
	FindByID(ctx context.Context, id string) (domain.Ad, error)
}

type AdHandler struct {
	ads AdService
}

func NewAdHandler(ads AdService) *AdHandler {
	return &AdHandler{ads: ads}
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
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

	writeJSON(w, http.StatusCreated, dto.NewAdResponse(ad))
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
