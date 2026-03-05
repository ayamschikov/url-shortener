package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ayamschikov/url-shortener/internal/model"
)

type ClickRepository struct {
	db *pgxpool.Pool
}

func NewClickRepository(db *pgxpool.Pool) *ClickRepository {
	return &ClickRepository{db: db}
}

func (r *ClickRepository) Save(ctx context.Context, click *model.Click) error {
	query := `
		INSERT INTO clicks (url_id, ip, user_agent, referer)
		VALUES ($1, $2, $3, $4)
		RETURNING id, clicked_at
	`
	return r.db.QueryRow(ctx, query, click.URLID, click.IP, click.UserAgent, click.Referer).
		Scan(&click.ID, &click.ClickedAt)
}

func (r *ClickRepository) GetStatsByURLID(ctx context.Context, urlID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM clicks WHERE url_id = $1`
	var count int64
	err := r.db.QueryRow(ctx, query, urlID).Scan(&count)
	return count, err
}
