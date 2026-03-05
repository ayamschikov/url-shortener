package service

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/ayamschikov/url-shortener/internal/model"
)

const (
	codeLength = 8
	alphabet   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var ErrURLExpired = errors.New("url has expired")

type URLRepository interface {
	Save(ctx context.Context, url *model.URL) error
	FindByCode(ctx context.Context, code string) (*model.URL, error)
}

type URLCache interface {
	Get(ctx context.Context, code string) (string, error)
	Set(ctx context.Context, code string, originalURL string) error
}

type ClickRepository interface {
	Save(ctx context.Context, click *model.Click) error
	GetStatsByURLID(ctx context.Context, urlID int64) (int64, error)
}

type URLService struct {
	repo   URLRepository
	cache  URLCache
	clicks ClickRepository
}

func NewURLService(repo URLRepository, cache URLCache, clicks ClickRepository) *URLService {
	return &URLService{repo: repo, cache: cache, clicks: clicks}
}

func (s *URLService) Shorten(ctx context.Context, originalURL string) (*model.URL, error) {
	code, err := generateCode()
	if err != nil {
		return nil, err
	}

	url := &model.URL{
		Code:        code,
		OriginalURL: originalURL,
	}

	if err := s.repo.Save(ctx, url); err != nil {
		return nil, err
	}

	return url, nil
}

func (s *URLService) Resolve(ctx context.Context, code string) (*model.URL, error) {
	// 1. Проверяем кеш
	if cached, err := s.cache.Get(ctx, code); err == nil {
		return &model.URL{Code: code, OriginalURL: cached}, nil
	}

	// 2. Не в кеше — идём в БД
	url, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if url.ExpiresAt != nil && url.ExpiresAt.Before(time.Now()) {
		return nil, ErrURLExpired
	}

	// 3. Сохраняем в кеш (ошибку кеша игнорируем — не критично)
	s.cache.Set(ctx, code, url.OriginalURL)

	return url, nil
}

func (s *URLService) TrackClick(ctx context.Context, click *model.Click) {
	// Запускаем в отдельной goroutine чтобы не замедлять редирект.
	// Используем context.Background() потому что оригинальный ctx
	// будет отменён после завершения HTTP ответа.
	go func() {
		s.clicks.Save(context.Background(), click)
	}()
}

func (s *URLService) GetStats(ctx context.Context, code string) (*model.URLStats, error) {
	url, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	totalClicks, err := s.clicks.GetStatsByURLID(ctx, url.ID)
	if err != nil {
		return nil, err
	}

	return &model.URLStats{
		Code:        url.Code,
		OriginalURL: url.OriginalURL,
		TotalClicks: totalClicks,
	}, nil
}

func generateCode() (string, error) {
	result := make([]byte, codeLength)
	alphabetLen := big.NewInt(int64(len(alphabet)))

	for i := range result {
		n, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		result[i] = alphabet[n.Int64()]
	}

	return string(result), nil
}
