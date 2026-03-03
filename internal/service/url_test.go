package service

import (
	"context"
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

func TestShorten_Success(t *testing.T) {
	repo := &mockRepo{
		saveFunc: func(ctx context.Context, url *model.URL) error {
			url.ID = 1
			url.CreatedAt = time.Now()
			return nil
		},
	}

	svc := NewURLService(repo)
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

	svc := NewURLService(repo)
	originalURL, err := svc.Resolve(context.Background(), "abc12345")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if originalURL != "https://google.com" {
		t.Errorf("expected https://google.com, got %s", originalURL)
	}
}

func TestResolve_NotFound(t *testing.T) {
	repo := &mockRepo{
		findByCodeFunc: func(ctx context.Context, code string) (*model.URL, error) {
			return nil, repository.ErrNotFound
		},
	}

	svc := NewURLService(repo)
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

	svc := NewURLService(repo)
	_, err := svc.Resolve(context.Background(), "expired1")

	if err != ErrURLExpired {
		t.Errorf("expected ErrURLExpired, got %v", err)
	}
}
