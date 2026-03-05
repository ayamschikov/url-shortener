package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ayamschikov/url-shortener/internal/model"
	"github.com/ayamschikov/url-shortener/internal/repository"
)

// mockRepo — мок репозитория для тестов.
// В Go не нужен отдельный фреймворк для моков —
// достаточно реализовать интерфейс вручную.
type mockRepo struct {
	saveFunc       func(ctx context.Context, url *model.URL) error
	findByCodeFunc func(ctx context.Context, code string) (*model.URL, error)
}

func (m *mockRepo) Save(ctx context.Context, url *model.URL) error {
	return m.saveFunc(ctx, url)
}

func (m *mockRepo) FindByCode(ctx context.Context, code string) (*model.URL, error) {
	return m.findByCodeFunc(ctx, code)
}

type mockCache struct {
	getFunc func(ctx context.Context, code string) (string, error)
	setFunc func(ctx context.Context, code string, originalURL string) error
}

func (m *mockCache) Get(ctx context.Context, code string) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, code)
	}
	return "", errors.New("cache miss")
}

func (m *mockCache) Set(ctx context.Context, code string, originalURL string) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, code, originalURL)
	}
	return nil
}

type mockClickRepo struct{}

func (m *mockClickRepo) Save(ctx context.Context, click *model.Click) error {
	return nil
}

func (m *mockClickRepo) GetStatsByURLID(ctx context.Context, urlID int64) (int64, error) {
	return 0, nil
}

func TestShorten_Success(t *testing.T) {
	repo := &mockRepo{
		saveFunc: func(ctx context.Context, url *model.URL) error {
			url.ID = 1
			url.CreatedAt = time.Now()
			return nil
		},
	}

	svc := NewURLService(repo, &mockCache{}, &mockClickRepo{})
	url, err := svc.Shorten(context.Background(), "https://google.com")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url.OriginalURL != "https://google.com" {
		t.Errorf("expected original url https://google.com, got %s", url.OriginalURL)
	}
	if len(url.Code) != codeLength {
		t.Errorf("expected code length %d, got %d", codeLength, len(url.Code))
	}
}

func TestResolve_Success(t *testing.T) {
	repo := &mockRepo{
		findByCodeFunc: func(ctx context.Context, code string) (*model.URL, error) {
			return &model.URL{
				ID:          1,
				Code:        code,
				OriginalURL: "https://google.com",
				CreatedAt:   time.Now(),
			}, nil
		},
	}

	svc := NewURLService(repo, &mockCache{}, &mockClickRepo{})
	url, err := svc.Resolve(context.Background(), "abc12345")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url.OriginalURL != "https://google.com" {
		t.Errorf("expected https://google.com, got %s", url.OriginalURL)
	}
}

func TestResolve_NotFound(t *testing.T) {
	repo := &mockRepo{
		findByCodeFunc: func(ctx context.Context, code string) (*model.URL, error) {
			return nil, repository.ErrNotFound
		},
	}

	svc := NewURLService(repo, &mockCache{}, &mockClickRepo{})
	_, err := svc.Resolve(context.Background(), "notexist")

	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestResolve_Expired(t *testing.T) {
	expired := time.Now().Add(-1 * time.Hour)
	repo := &mockRepo{
		findByCodeFunc: func(ctx context.Context, code string) (*model.URL, error) {
			return &model.URL{
				ID:          1,
				Code:        code,
				OriginalURL: "https://google.com",
				ExpiresAt:   &expired,
			}, nil
		},
	}

	svc := NewURLService(repo, &mockCache{}, &mockClickRepo{})
	_, err := svc.Resolve(context.Background(), "expired1")

	if err != ErrURLExpired {
		t.Errorf("expected ErrURLExpired, got %v", err)
	}
}

func TestResolve_CacheHit(t *testing.T) {
	dbCalled := false
	repo := &mockRepo{
		findByCodeFunc: func(ctx context.Context, code string) (*model.URL, error) {
			dbCalled = true
			return nil, errors.New("should not be called")
		},
	}
	c := &mockCache{
		getFunc: func(ctx context.Context, code string) (string, error) {
			return "https://google.com", nil
		},
	}

	svc := NewURLService(repo, c, &mockClickRepo{})
	url, err := svc.Resolve(context.Background(), "cached1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url.OriginalURL != "https://google.com" {
		t.Errorf("expected https://google.com, got %s", url.OriginalURL)
	}
	if dbCalled {
		t.Error("expected database NOT to be called when cache hits")
	}
}

func TestResolve_CacheMiss_ThenSetsCache(t *testing.T) {
	cached := false
	repo := &mockRepo{
		findByCodeFunc: func(ctx context.Context, code string) (*model.URL, error) {
			return &model.URL{
				ID:          1,
				Code:        code,
				OriginalURL: "https://google.com",
			}, nil
		},
	}
	c := &mockCache{
		setFunc: func(ctx context.Context, code string, originalURL string) error {
			cached = true
			return nil
		},
	}

	svc := NewURLService(repo, c, &mockClickRepo{})
	_, err := svc.Resolve(context.Background(), "notcached")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cached {
		t.Error("expected cache.Set to be called after DB lookup")
	}
}
