package service

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/ayamschikov/url-shortener/internal/model"
	"github.com/ayamschikov/url-shortener/internal/repository"
)

const (
	codeLength = 8
	alphabet   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var ErrURLExpired = errors.New("url has expired")

type URLService struct {
	repo *repository.URLRepository
}

func NewURLService(repo *repository.URLRepository) *URLService {
	return &URLService{repo: repo}
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

func (s *URLService) Resolve(ctx context.Context, code string) (string, error) {
	url, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return "", err
	}

	if url.ExpiresAt != nil && url.ExpiresAt.Before(time.Now()) {
		return "", ErrURLExpired
	}

	return url.OriginalURL, nil
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
