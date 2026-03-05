package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ayamschikov/url-shortener/internal/model"
	"github.com/ayamschikov/url-shortener/internal/repository"
	"github.com/ayamschikov/url-shortener/internal/service"
)

type mockService struct {
	shortenFunc    func(ctx context.Context, originalURL string, alias string, expiresAt *time.Time) (*model.URL, error)
	resolveFunc    func(ctx context.Context, code string) (*model.URL, error)
	trackClickFunc func(ctx context.Context, click *model.Click)
}

func (m *mockService) Shorten(ctx context.Context, originalURL string, alias string, expiresAt *time.Time) (*model.URL, error) {
	return m.shortenFunc(ctx, originalURL, alias, expiresAt)
}

func (m *mockService) Resolve(ctx context.Context, code string) (*model.URL, error) {
	return m.resolveFunc(ctx, code)
}

func (m *mockService) TrackClick(ctx context.Context, click *model.Click) {
	if m.trackClickFunc != nil {
		m.trackClickFunc(ctx, click)
	}
}

func (m *mockService) GetStats(ctx context.Context, code string) (*model.URLStats, error) {
	return &model.URLStats{Code: code, OriginalURL: "https://google.com", TotalClicks: 42}, nil
}

func TestShorten_Success(t *testing.T) {
	svc := &mockService{
		shortenFunc: func(ctx context.Context, originalURL string, alias string, expiresAt *time.Time) (*model.URL, error) {
			return &model.URL{Code: "abc12345", OriginalURL: originalURL, ExpiresAt: expiresAt}, nil
		},
	}
	h := NewURLHandler(svc)

	body := strings.NewReader(`{"url": "https://google.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	rec := httptest.NewRecorder()

	h.Shorten(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "abc12345") {
		t.Errorf("expected response to contain code, got %s", rec.Body.String())
	}
}

func TestShorten_EmptyURL(t *testing.T) {
	h := NewURLHandler(&mockService{})

	body := strings.NewReader(`{"url": ""}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	rec := httptest.NewRecorder()

	h.Shorten(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestShorten_InvalidBody(t *testing.T) {
	h := NewURLHandler(&mockService{})

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	rec := httptest.NewRecorder()

	h.Shorten(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestShorten_ServiceError(t *testing.T) {
	svc := &mockService{
		shortenFunc: func(ctx context.Context, originalURL string, alias string, expiresAt *time.Time) (*model.URL, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewURLHandler(svc)

	body := strings.NewReader(`{"url": "https://google.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	rec := httptest.NewRecorder()

	h.Shorten(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// newResolveRequest создаёт запрос с chi URL параметром.
// chi.URLParam читает параметры из контекста запроса,
// поэтому нужно их туда положить вручную.
func newResolveRequest(code string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", code)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestResolve_Success(t *testing.T) {
	svc := &mockService{
		resolveFunc: func(ctx context.Context, code string) (*model.URL, error) {
			return &model.URL{ID: 1, Code: code, OriginalURL: "https://google.com"}, nil
		},
	}
	h := NewURLHandler(svc)

	req := newResolveRequest("abc12345")
	rec := httptest.NewRecorder()

	h.Resolve(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
	}
	location := rec.Header().Get("Location")
	if location != "https://google.com" {
		t.Errorf("expected redirect to https://google.com, got %s", location)
	}
}

func TestResolve_NotFound(t *testing.T) {
	svc := &mockService{
		resolveFunc: func(ctx context.Context, code string) (*model.URL, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewURLHandler(svc)

	req := newResolveRequest("notexist")
	rec := httptest.NewRecorder()

	h.Resolve(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestResolve_Expired(t *testing.T) {
	svc := &mockService{
		resolveFunc: func(ctx context.Context, code string) (*model.URL, error) {
			return nil, service.ErrURLExpired
		},
	}
	h := NewURLHandler(svc)

	req := newResolveRequest("expired1")
	rec := httptest.NewRecorder()

	h.Resolve(rec, req)

	if rec.Code != http.StatusGone {
		t.Errorf("expected status %d, got %d", http.StatusGone, rec.Code)
	}
}
