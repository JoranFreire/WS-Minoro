package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/ws-minoro/link-router/internal/cache"
)

type Config struct {
	MaxRequests int
	WindowSecs  int
}

type RateLimiter struct {
	cache  *cache.RedisCache
	config Config
}

func NewRateLimiter(c *cache.RedisCache, cfg Config) *RateLimiter {
	return &RateLimiter{cache: c, config: cfg}
}

func (r *RateLimiter) Allow(ctx context.Context, ip string) (bool, error) {
	key := fmt.Sprintf("rl:%s", ip)
	ttl := time.Duration(r.config.WindowSecs) * time.Second

	count, err := r.cache.IncrWithExpiry(ctx, key, ttl)
	if err != nil {
		// Fail open on Redis error
		return true, err
	}

	return count <= int64(r.config.MaxRequests), nil
}
