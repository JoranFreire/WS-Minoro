package quota

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestTracker(t *testing.T) (*Tracker, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewTrackerWithClient(client), client
}

func TestMarkOverQuota_SetsTheBlockedKey(t *testing.T) {
	tr, client := newTestTracker(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)

	if err := tr.MarkOverQuota(ctx, "tenant-1", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := client.Get(ctx, "quota:blocked:tenant-1").Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "1" {
		t.Fatalf("expected value 1, got %q", val)
	}
}

func TestMarkOverQuota_ExpiresAtStartOfNextMonth(t *testing.T) {
	tr, client := newTestTracker(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)

	if err := tr.MarkOverQuota(ctx, "tenant-1", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ttl, err := client.TTL(ctx, "quota:blocked:tenant-1").Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantMax := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).Sub(now)
	if ttl <= 0 || ttl > wantMax {
		t.Fatalf("expected TTL between 0 and %v, got %v", wantMax, ttl)
	}
}

func TestMarkOverQuota_ScopedPerTenant(t *testing.T) {
	tr, client := newTestTracker(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)

	if err := tr.MarkOverQuota(ctx, "tenant-1", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := client.Get(ctx, "quota:blocked:tenant-2").Result(); err == nil {
		t.Fatal("expected tenant-2 to not be blocked")
	}
}
