package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ayamschikov/url-shortener/internal/model"
)

var ErrNotFound = errors.New("url not found")

type URLRepository struct {
	db *pgxpool.Pool
}

func NewURLRepository(db *pgxpool.Pool) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Save(ctx context.Context, url *model.URL) error {
	query := `
		INSERT INTO urls (code, original_url, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query, url.Code, url.OriginalURL, url.ExpiresAt).
		Scan(&url.ID, &url.CreatedAt)
}

func (r *URLRepository) FindByCode(ctx context.Context, code string) (*model.URL, error) {
	query := `
		SELECT id, code, original_url, created_at, expires_at
		FROM urls
		WHERE code = $1
	`
	url := &model.URL{}
	err := r.db.QueryRow(ctx, query, code).
		Scan(&url.ID, &url.Code, &url.OriginalURL, &url.CreatedAt, &url.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return url, nil
}
