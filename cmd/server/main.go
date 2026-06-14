package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/ayamschikov/url-shortener/internal/cache"
	"github.com/ayamschikov/url-shortener/internal/handler"
	appmiddleware "github.com/ayamschikov/url-shortener/internal/middleware"
	"github.com/ayamschikov/url-shortener/internal/repository"
	"github.com/ayamschikov/url-shortener/internal/service"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL is not set")
	}

	db, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("failed to parse redis url: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	repo := repository.NewURLRepository(db)
	clickRepo := repository.NewClickRepository(db)
	urlCache := cache.NewURLCache(redisClient)
	svc := service.NewURLService(repo, urlCache, clickRepo)
	h := handler.NewURLHandler(svc)

	// Rate limit is configurable so a load test can exceed the default.
	rateLimit := 100
	if v := os.Getenv("RATE_LIMIT_PER_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rateLimit = n
		}
	}
	rateLimiter := appmiddleware.NewRateLimiter(redisClient, rateLimit, time.Minute)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(appmiddleware.Metrics)
	r.Use(rateLimiter.Middleware)

	// Prometheus scrape target. No auth, not rate-limited in practice
	// (scraped every 15s from one source).
	r.Handle("/metrics", promhttp.Handler())

	r.Post("/shorten", h.Shorten)
	r.Get("/stats/{code}", h.Stats)
	r.Get("/{code}", h.Resolve)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
