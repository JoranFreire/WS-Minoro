package health

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ws-minoro/link-router/internal/cache"
	"github.com/ws-minoro/link-router/internal/event"
	"github.com/ws-minoro/link-router/internal/store"
)

// InviteHealthMonitor periodically scans for destinations whose cooldown has
// expired and reactivates them, flushing their Redis cache entry.
type InviteHealthMonitor struct {
	store     *store.PGStore
	cache     *cache.RedisCache
	publisher *event.HealthPublisher
	interval  time.Duration
}

func NewInviteHealthMonitor(
	s *store.PGStore,
	c *cache.RedisCache,
	p *event.HealthPublisher,
	intervalSec int,
) *InviteHealthMonitor {
	return &InviteHealthMonitor{
		store:     s,
		cache:     c,
		publisher: p,
		interval:  time.Duration(intervalSec) * time.Second,
	}
}

// Start runs the monitor loop until the context is cancelled.
func (m *InviteHealthMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	log.Printf("invite_health_monitor: started (interval=%s)", m.interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("invite_health_monitor: stopped")
			return
		case <-ticker.C:
			m.run(ctx)
		}
	}
}

func (m *InviteHealthMonitor) run(ctx context.Context) {
	expired, err := m.store.GetExpiredCooldownDestinations(ctx)
	if err != nil {
		log.Printf("invite_health_monitor: query error: %v", err)
		return
	}

	for _, d := range expired {
		if err := m.store.ReactivateDestination(ctx, d.ID); err != nil {
			log.Printf("invite_health_monitor: reactivate %s error: %v", d.ID, err)
			continue
		}

		// Invalidate the route cache so the next request re-fetches from DB.
		cacheKey := fmt.Sprintf("route:%s", d.ShortCode)
		_ = m.cache.Delete(ctx, cacheKey)

		m.publisher.PublishReactivated(ctx, event.HealthEvent{
			DestinationID: d.ID,
			LinkID:        d.LinkID,
			TenantID:      d.TenantID,
			ShortCode:     d.ShortCode,
			Reason:        "cooldown_expired",
			Timestamp:     time.Now().UTC(),
		})

		log.Printf("invite_health_monitor: reactivated destination %s (link=%s)", d.ID, d.LinkID)
	}
}
