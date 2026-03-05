package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ayamschikov/url-shortener/internal/httputil"
	"github.com/ayamschikov/url-shortener/internal/model"
	"github.com/ayamschikov/url-shortener/internal/repository"
	"github.com/ayamschikov/url-shortener/internal/service"
)

type URLService interface {
	Shorten(ctx context.Context, originalURL string) (*model.URL, error)
	Resolve(ctx context.Context, code string) (*model.URL, error)
	TrackClick(ctx context.Context, click *model.Click)
	GetStats(ctx context.Context, code string) (*model.URLStats, error)
}

type URLHandler struct {
	service URLService
}

func NewURLHandler(service URLService) *URLHandler {
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

	url, err := h.service.Resolve(r.Context(), code)
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

	h.service.TrackClick(r.Context(), &model.Click{
		URLID:     url.ID,
		IP:        httputil.ExtractIP(r),
		UserAgent: r.Header.Get("User-Agent"),
		Referer:   r.Header.Get("Referer"),
	})

	http.Redirect(w, r, url.OriginalURL, http.StatusMovedPermanently)
}

type statsResponse struct {
	Code        string `json:"code"`
	OriginalURL string `json:"original_url"`
	TotalClicks int64  `json:"total_clicks"`
}

func (h *URLHandler) Stats(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	stats, err := h.service.GetStats(r.Context(), code)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "url not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, statsResponse{
		Code:        stats.Code,
		OriginalURL: stats.OriginalURL,
		TotalClicks: stats.TotalClicks,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
