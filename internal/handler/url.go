package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ayamschikov/url-shortener/internal/httputil"
	"github.com/ayamschikov/url-shortener/internal/model"
	"github.com/ayamschikov/url-shortener/internal/repository"
	"github.com/ayamschikov/url-shortener/internal/service"
)

type URLService interface {
	Shorten(ctx context.Context, originalURL string, alias string, expiresAt *time.Time) (*model.URL, error)
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
	URL       string `json:"url"`
	Alias     string `json:"alias,omitempty"`
	ExpiresIn string `json:"expires_in,omitempty"`
}

type shortenResponse struct {
	ShortURL  string  `json:"short_url"`
	ExpiresAt *string `json:"expires_at,omitempty"`
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

	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		duration, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid expires_in format, use e.g. 1h, 24h, 168h"})
			return
		}
		t := time.Now().Add(duration)
		expiresAt = &t
	}

	url, err := h.service.Shorten(r.Context(), req.URL, req.Alias, expiresAt)
	if errors.Is(err, service.ErrAliasInvalid) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if errors.Is(err, service.ErrAliasTaken) {
		writeJSON(w, http.StatusConflict, errorResponse{Error: "alias already taken"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to shorten url"})
		return
	}

	resp := shortenResponse{ShortURL: url.Code}
	if url.ExpiresAt != nil {
		formatted := url.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &formatted
	}
	writeJSON(w, http.StatusCreated, resp)
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
