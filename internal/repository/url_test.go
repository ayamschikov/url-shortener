package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ayamschikov/url-shortener/internal/model"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("urlshortener_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		pgContainer.Terminate(ctx)
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Применяем миграции через goose
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to open db for migrations: %v", err)
	}
	defer db.Close()

	if err := goose.Up(db, "../../migrations"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Создаём pgxpool для repository
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func TestSave_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewURLRepository(pool)

	url := &model.URL{
		Code:        "test1234",
		OriginalURL: "https://google.com",
	}

	err := repo.Save(context.Background(), url)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url.ID == 0 {
		t.Error("expected ID to be set")
	}
	if url.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestSave_DuplicateCode(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewURLRepository(pool)

	url1 := &model.URL{Code: "dupcode1", OriginalURL: "https://google.com"}
	url2 := &model.URL{Code: "dupcode1", OriginalURL: "https://github.com"}

	if err := repo.Save(context.Background(), url1); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	err := repo.Save(context.Background(), url2)
	if err == nil {
		t.Error("expected error on duplicate code, got nil")
	}
}

func TestFindByCode_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewURLRepository(pool)

	original := &model.URL{Code: "find1234", OriginalURL: "https://google.com"}
	repo.Save(context.Background(), original)

	found, err := repo.FindByCode(context.Background(), "find1234")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.OriginalURL != "https://google.com" {
		t.Errorf("expected https://google.com, got %s", found.OriginalURL)
	}
	if found.Code != "find1234" {
		t.Errorf("expected code find1234, got %s", found.Code)
	}
}

func TestFindByCode_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewURLRepository(pool)

	_, err := repo.FindByCode(context.Background(), "notexist")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
