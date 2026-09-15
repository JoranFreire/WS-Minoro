// Package quota clears the "over quota" flag link-router checks on the
// redirect hot path (set by analytics-worker in internal/quota there).
// Without this, a tenant who just paid to upgrade would stay blocked until
// the flag's TTL expires at month's end.
package quota

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Unblocker struct {
	client *redis.Client
}

func NewUnblocker(redisURL string) *Unblocker {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic("invalid redis URL: " + err.Error())
	}
	return &Unblocker{client: redis.NewClient(opt)}
}

// NewUnblockerWithClient wraps an existing redis client, allowing tests to
// inject a fake/in-memory client instead of dialing a real server.
func NewUnblockerWithClient(client *redis.Client) *Unblocker {
	return &Unblocker{client: client}
}

// Clear removes the over-quota flag for a tenant immediately, instead of
// waiting for it to expire at month's end.
func (u *Unblocker) Clear(ctx context.Context, tenantID string) error {
	return u.client.Del(ctx, "quota:blocked:"+tenantID).Err()
}
