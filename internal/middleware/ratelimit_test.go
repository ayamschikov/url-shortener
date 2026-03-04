package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6397"
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("failed to parse REDIS_URL: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() {
		client.FlushDB(context.Background())
		client.Close()
	})
	return client
}

// okHandler — простой handler который всегда отвечает 200
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	client := setupTestRedis(t)
	rl := NewRateLimiter(client, 5, time.Minute)
	handler := rl.Middleware(okHandler())

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	client := setupTestRedis(t)
	rl := NewRateLimiter(client, 3, time.Minute)
	handler := rl.Middleware(okHandler())

	// Первые 3 — проходят
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// 4-й — блокируется
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	client := setupTestRedis(t)
	rl := NewRateLimiter(client, 1, time.Minute)
	handler := rl.Middleware(okHandler())

	// IP 1 — первый запрос проходит
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.1.1.1:12345"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// IP 2 — тоже проходит (отдельный лимит)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "2.2.2.2:12345"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec1.Code != http.StatusOK {
		t.Errorf("IP 1: expected 200, got %d", rec1.Code)
	}
	if rec2.Code != http.StatusOK {
		t.Errorf("IP 2: expected 200, got %d", rec2.Code)
	}
}
