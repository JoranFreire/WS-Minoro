package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ws-minoro/link-router/internal/cache"
	"github.com/ws-minoro/link-router/internal/event"
	"github.com/ws-minoro/link-router/internal/risk"
	"github.com/ws-minoro/link-router/internal/router"
	"github.com/ws-minoro/link-router/internal/store"
)

const routeCacheTTL = 60 * time.Second

var ErrNotFound = errors.New("route not found")

type LinkResolver struct {
	cache        *cache.RedisCache
	store        *store.PGStore
	healthPub    *event.HealthPublisher
	maxRiskScore float64
}

func NewLinkResolver(
	c *cache.RedisCache,
	s *store.PGStore,
	hp *event.HealthPublisher,
	maxRiskScore float64,
) *LinkResolver {
	return &LinkResolver{
		cache:        c,
		store:        s,
		healthPub:    hp,
		maxRiskScore: maxRiskScore,
	}
}

// Resolve returns the destination URL for the given short code, applying
// Phase 2 intelligent filtering (cooldown, risk score, max_clicks).
func (r *LinkResolver) Resolve(ctx context.Context, shortCode string) (destURL, linkID, tenantID string, err error) {
	route, err := r.getRoute(ctx, shortCode)
	if err != nil {
		return "", "", "", ErrNotFound
	}

	active := r.filterDestinations(route.Destinations)
	if len(active) == 0 {
		if route.FallbackURL != "" {
			return route.FallbackURL, route.LinkID, route.TenantID, nil
		}
		return "", "", "", ErrNotFound
	}

	var dest store.Destination
	switch route.RoutingStrategy {
	case "weighted":
		dest = router.SelectWeighted(active)
	default:
		dest, err = router.SelectRoundRobin(ctx, r.cache, route.LinkID, active)
		if err != nil {
			dest = active[0]
		}
	}

	// Track the click asynchronously — do not block the redirect.
	go r.trackClick(route, dest)

	return dest.URL, route.LinkID, route.TenantID, nil
}

// filterDestinations removes destinations that are in cooldown or exceed the risk threshold.
func (r *LinkResolver) filterDestinations(dests []store.Destination) []store.Destination {
	now := time.Now().UTC()
	active := make([]store.Destination, 0, len(dests))
	for _, d := range dests {
		if d.CooldownUntil != nil && now.Before(*d.CooldownUntil) {
			continue
		}
		if risk.IsRisky(d, r.maxRiskScore) {
			continue
		}
		active = append(active, d)
	}
	return active
}

// trackClick increments click count and disables the destination if max_clicks is reached.
func (r *LinkResolver) trackClick(route *store.RouteData, dest store.Destination) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.store.IncrDestinationClicks(ctx, dest.ID)
	if err != nil {
		log.Printf("resolver: incr clicks for %s: %v", dest.ID, err)
		return
	}

	if result.MaxClicks != nil && result.CurrentClicks >= *result.MaxClicks {
		if disableErr := r.store.DisableDestination(ctx, dest.ID); disableErr != nil {
			log.Printf("resolver: disable destination %s: %v", dest.ID, disableErr)
			return
		}

		// Bust the route cache so the next request sees the updated state.
		cacheKey := fmt.Sprintf("route:%s", route.ShortCode)
		_ = r.cache.Delete(ctx, cacheKey)

		r.healthPub.PublishDisabled(ctx, event.HealthEvent{
			DestinationID: dest.ID,
			LinkID:        route.LinkID,
			TenantID:      route.TenantID,
			ShortCode:     route.ShortCode,
			Reason:        "max_clicks_reached",
			Timestamp:     time.Now().UTC(),
		})

		log.Printf("resolver: destination %s auto-disabled (max_clicks=%d)", dest.ID, *result.MaxClicks)
	}
}

func (r *LinkResolver) getRoute(ctx context.Context, shortCode string) (*store.RouteData, error) {
	cacheKey := fmt.Sprintf("route:%s", shortCode)

	cached, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var route store.RouteData
		if jsonErr := json.Unmarshal([]byte(cached), &route); jsonErr == nil {
			return &route, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		// Cache error — fall through to DB.
	}

	route, dbErr := r.store.GetRouteByShortCode(ctx, shortCode)
	if dbErr != nil {
		return nil, ErrNotFound
	}

	if data, jsonErr := json.Marshal(route); jsonErr == nil {
		_ = r.cache.Set(ctx, cacheKey, string(data), routeCacheTTL)
	}

	return route, nil
}
