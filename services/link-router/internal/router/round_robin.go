package router

import (
	"context"
	"fmt"

	"github.com/ws-minoro/link-router/internal/cache"
	"github.com/ws-minoro/link-router/internal/store"
)

func SelectRoundRobin(ctx context.Context, c *cache.RedisCache, linkID string, destinations []store.Destination) (store.Destination, error) {
	key := fmt.Sprintf("rr_cursor:%s", linkID)
	idx, err := c.Incr(ctx, key)
	if err != nil {
		return destinations[0], err
	}
	return destinations[(idx-1)%int64(len(destinations))], nil
}
