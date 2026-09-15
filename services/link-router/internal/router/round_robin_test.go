package router

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/ws-minoro/link-router/internal/cache"
	"github.com/ws-minoro/link-router/internal/store"
)

func newTestCache(t *testing.T) *cache.RedisCache {
	t.Helper()
	mr := miniredis.RunT(t)
	return cache.NewRedisCacheWithClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
}

func TestSelectRoundRobin_CyclesThroughDestinationsInOrder(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	dests := []store.Destination{{ID: "a"}, {ID: "b"}, {ID: "c"}}

	want := []string{"a", "b", "c", "a", "b", "c"}
	for i, w := range want {
		got, err := SelectRoundRobin(ctx, c, "link-1", dests)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if got.ID != w {
			t.Fatalf("call %d: expected %s, got %s", i, w, got.ID)
		}
	}
}

func TestSelectRoundRobin_CursorIsIsolatedPerLink(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	dests := []store.Destination{{ID: "a"}, {ID: "b"}}

	got1, err := SelectRoundRobin(ctx, c, "link-1", dests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got2, err := SelectRoundRobin(ctx, c, "link-2", dests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got1.ID != "a" || got2.ID != "a" {
		t.Fatalf("expected both links to start at their own first destination, got %s and %s", got1.ID, got2.ID)
	}
}
