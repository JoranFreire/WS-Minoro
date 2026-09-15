package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestAggregator(t *testing.T) (*RedisAggregator, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewRedisAggregatorWithClient(client), client
}

func TestIncrClick_IncrementsHourDayAndTenantCounters(t *testing.T) {
	agg, client := newTestAggregator(t)
	ctx := context.Background()
	ts := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)

	if err := agg.IncrClick(ctx, "link-1", "tenant-1", ts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hourKey := "agg:link:link-1:hour:2026030514"
	dayKey := "agg:link:link-1:day:20260305"
	tenantKey := "agg:tenant:tenant-1:day:20260305"

	for _, key := range []string{hourKey, dayKey, tenantKey} {
		val, err := client.Get(ctx, key).Result()
		if err != nil {
			t.Fatalf("key %s: unexpected error: %v", key, err)
		}
		if val != "1" {
			t.Fatalf("key %s: expected 1, got %s", key, val)
		}
	}
}

func TestIncrClick_AccumulatesAcrossCalls(t *testing.T) {
	agg, client := newTestAggregator(t)
	ctx := context.Background()
	ts := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)

	for range 3 {
		if err := agg.IncrClick(ctx, "link-1", "tenant-1", ts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	val, err := client.Get(ctx, "agg:link:link-1:day:20260305").Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "3" {
		t.Fatalf("expected 3, got %s", val)
	}
}

func TestIncrClick_SetsExpiryOnCounters(t *testing.T) {
	agg, client := newTestAggregator(t)
	ctx := context.Background()
	ts := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)

	if err := agg.IncrClick(ctx, "link-1", "tenant-1", ts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ttl, err := client.TTL(ctx, "agg:link:link-1:hour:2026030514").Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("expected hour counter to have a positive TTL, got %v", ttl)
	}
}
