package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client   *redis.Client
	limit    int
	window   time.Duration
}

func NewRateLimiter(client *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		key := fmt.Sprintf("ratelimit:%s", ip)

		allowed, err := rl.isAllowed(r.Context(), key)
		if err != nil {
			// Если Redis недоступен — пропускаем запрос.
			// Лучше работать без rate limit, чем отказывать всем.
			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limit exceeded"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) isAllowed(ctx context.Context, key string) (bool, error) {
	count, err := rl.client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// Первый запрос в окне — ставим TTL
	if count == 1 {
		rl.client.Expire(ctx, key, rl.window)
	}

	return count <= int64(rl.limit), nil
}

func extractIP(r *http.Request) string {
	// Сначала проверяем заголовки от прокси/балансировщика
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fallback — IP из соединения
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
