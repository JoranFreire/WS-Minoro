package ratelimit

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/ws-minoro/link-router/internal/cache"
)

func newTestLimiter(t *testing.T, cfg Config) *RateLimiter {
	t.Helper()
	mr := miniredis.RunT(t)
	c := cache.NewRedisCacheWithClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	return NewRateLimiter(c, cfg)
}

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := newTestLimiter(t, Config{MaxRequests: 3, WindowSecs: 60})
	ctx := context.Background()

	for i := range 3 {
		allowed, err := rl.Allow(ctx, "1.2.3.4")
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
		if !allowed {
			t.Fatalf("request %d: expected allowed", i)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := newTestLimiter(t, Config{MaxRequests: 2, WindowSecs: 60})
	ctx := context.Background()

	_, _ = rl.Allow(ctx, "1.2.3.4")
	_, _ = rl.Allow(ctx, "1.2.3.4")
	allowed, err := rl.Allow(ctx, "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("expected third request to be blocked")
	}
}

func TestRateLimiter_TracksIPsIndependently(t *testing.T) {
	rl := newTestLimiter(t, Config{MaxRequests: 1, WindowSecs: 60})
	ctx := context.Background()

	allowedA, _ := rl.Allow(ctx, "1.1.1.1")
	allowedB, _ := rl.Allow(ctx, "2.2.2.2")

	if !allowedA || !allowedB {
		t.Fatalf("expected both distinct IPs to be allowed independently, got a=%v b=%v", allowedA, allowedB)
	}
}
