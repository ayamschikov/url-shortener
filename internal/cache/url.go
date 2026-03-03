package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultTTL = 24 * time.Hour

type URLCache struct {
	client *redis.Client
}

func NewURLCache(client *redis.Client) *URLCache {
	return &URLCache{client: client}
}

func (c *URLCache) Get(ctx context.Context, code string) (string, error) {
	return c.client.Get(ctx, "url:"+code).Result()
}

func (c *URLCache) Set(ctx context.Context, code string, originalURL string) error {
	return c.client.Set(ctx, "url:"+code, originalURL, defaultTTL).Err()
}
