package quota

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Tracker flags tenants that have exceeded their monthly click quota, in
// Redis so link-router can check it on the hot redirect path without
// hitting Postgres per request. Both services point at the same Redis
// instance.
type Tracker struct {
	client *redis.Client
}

func NewTracker(redisURL string) *Tracker {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic("invalid redis URL: " + err.Error())
	}
	return &Tracker{client: redis.NewClient(opt)}
}

// NewTrackerWithClient wraps an existing redis client, allowing tests to
// inject a fake/in-memory client instead of dialing a real server.
func NewTrackerWithClient(client *redis.Client) *Tracker {
	return &Tracker{client: client}
}

func blockedKey(tenantID string) string {
	return "quota:blocked:" + tenantID
}

// MarkOverQuota flags a tenant as having exceeded its monthly click quota.
// The flag expires automatically at the start of next month, so no separate
// reset job is needed once the counter itself rolls over.
func (t *Tracker) MarkOverQuota(ctx context.Context, tenantID string, now time.Time) error {
	return t.client.Set(ctx, blockedKey(tenantID), "1", untilNextMonth(now)).Err()
}

func untilNextMonth(now time.Time) time.Duration {
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	return nextMonth.Sub(now)
}
