package aggregator

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisAggregator struct {
	client *redis.Client
}

func NewRedisAggregator(redisURL string) *RedisAggregator {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic("invalid redis URL: " + err.Error())
	}
	return &RedisAggregator{client: redis.NewClient(opt)}
}

func (a *RedisAggregator) IncrClick(ctx context.Context, linkID, tenantID string, ts time.Time) error {
	hourKey := fmt.Sprintf("agg:link:%s:hour:%s", linkID, ts.Format("2006010215"))
	dayKey := fmt.Sprintf("agg:link:%s:day:%s", linkID, ts.Format("20060102"))
	tenantKey := fmt.Sprintf("agg:tenant:%s:day:%s", tenantID, ts.Format("20060102"))

	pipe := a.client.Pipeline()
	pipe.Incr(ctx, hourKey)
	pipe.Expire(ctx, hourKey, 25*time.Hour)
	pipe.Incr(ctx, dayKey)
	pipe.Expire(ctx, dayKey, 8*24*time.Hour)
	pipe.Incr(ctx, tenantKey)
	pipe.Expire(ctx, tenantKey, 8*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}
