package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) *RedisCache {
	t.Helper()
	mr := miniredis.RunT(t)
	return NewRedisCacheWithClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
}

func TestRedisCache_SetGet(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "v" {
		t.Fatalf("expected v, got %v", got)
	}
}

func TestRedisCache_GetMissingKeyReturnsError(t *testing.T) {
	c := newTestCache(t)
	_, err := c.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestRedisCache_Delete(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	_ = c.Set(ctx, "k", "v", time.Minute)
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.Get(ctx, "k"); err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestRedisCache_Incr(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	for expected := int64(1); expected <= 3; expected++ {
		got, err := c.Incr(ctx, "counter")
		if err != nil {
			t.Fatalf("Incr: %v", err)
		}
		if got != expected {
			t.Fatalf("expected %d, got %d", expected, got)
		}
	}
}

func TestRedisCache_IncrWithExpiry(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	got, err := c.IncrWithExpiry(ctx, "windowed", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("IncrWithExpiry: %v", err)
	}
	if got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}

	got, err = c.IncrWithExpiry(ctx, "windowed", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("IncrWithExpiry: %v", err)
	}
	if got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
}
