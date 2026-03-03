package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ayamschikov/url-shortener/internal/repository"
	"github.com/ayamschikov/url-shortener/internal/service"
)

type URLHandler struct {
	service *service.URLService
}

func NewURLHandler(service *service.URLService) *URLHandler {
	return &URLHandler{service: service}
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	ShortURL string `json:"short_url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *URLHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "url is required"})
		return
	}

	url, err := h.service.Shorten(r.Context(), req.URL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to shorten url"})
		return
	}

	writeJSON(w, http.StatusCreated, shortenResponse{ShortURL: url.Code})
}

func (h *URLHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	originalURL, err := h.service.Resolve(r.Context(), code)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "url not found"})
		return
	}
	if errors.Is(err, service.ErrURLExpired) {
		writeJSON(w, http.StatusGone, errorResponse{Error: "url has expired"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
