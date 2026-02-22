package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ws-minoro/link-router/internal/breaker"
	"github.com/ws-minoro/link-router/internal/cache"
	"github.com/ws-minoro/link-router/internal/event"
	"github.com/ws-minoro/link-router/internal/geo"
	"github.com/ws-minoro/link-router/internal/risk"
	"github.com/ws-minoro/link-router/internal/router"
	"github.com/ws-minoro/link-router/internal/store"
)

const routeCacheTTL = 60 * time.Second

var ErrNotFound = errors.New("route not found")

type LinkResolver struct {
	cache         *cache.RedisCache
	store         *store.PGStore
	healthPub     *event.HealthPublisher
	maxRiskScore  float64
	defaultDomain string
	redisBreaker  *breaker.CircuitBreaker
	pgBreaker     *breaker.CircuitBreaker
}

func NewLinkResolver(
	c *cache.RedisCache,
	s *store.PGStore,
	hp *event.HealthPublisher,
	maxRiskScore float64,
	defaultDomain string,
) *LinkResolver {
	return &LinkResolver{
		cache:         c,
		store:         s,
		healthPub:     hp,
		maxRiskScore:  maxRiskScore,
		defaultDomain: defaultDomain,
		// Open after 5 consecutive failures; retry after 10 s.
		redisBreaker: breaker.New(5, 10*time.Second),
		pgBreaker:    breaker.New(3, 15*time.Second),
	}
}

// Resolve selects a destination URL applying Phase 2 and Phase 6 logic.
// Returns (destURL, linkID, tenantID, experimentID, error).
func (r *LinkResolver) Resolve(
	ctx context.Context,
	shortCode, host, country string,
) (destURL, linkID, tenantID, experimentID string, err error) {
	route, err := r.getRoute(ctx, shortCode, host)
	if err != nil {
		return "", "", "", "", ErrNotFound
	}

	// Phase 2: filter out cooldown and high-risk destinations.
	active := r.filterDestinations(route.Destinations)

	// Phase 6: filter by country.
	active = geo.FilterByCountry(active, country)

	if len(active) == 0 {
		if route.FallbackURL != "" {
			return route.FallbackURL, route.LinkID, route.TenantID, "", nil
		}
		return "", "", "", "", ErrNotFound
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

	// Track click asynchronously — never block the redirect.
	go r.trackClick(route, dest)

	return dest.URL, route.LinkID, route.TenantID, dest.ExperimentID, nil
}

// filterDestinations removes Phase 2 ineligible destinations.
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

// trackClick increments click count and auto-disables when max_clicks is reached.
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

// getRoute fetches a route from cache (Redis) with a PG fallback.
// Phase 6: if host differs from the default domain, resolves via custom domain tenant scoping.
func (r *LinkResolver) getRoute(ctx context.Context, shortCode, host string) (*store.RouteData, error) {
	// Determine cache key — scoped by host for white-label domains.
	cacheKey := fmt.Sprintf("route:%s", shortCode)
	if host != "" && host != r.defaultDomain {
		cacheKey = fmt.Sprintf("route:%s:%s", host, shortCode)
	}

	// Try Redis cache through circuit breaker.
	var cached string
	_ = r.redisBreaker.Call(func() error {
		var e error
		cached, e = r.cache.Get(ctx, cacheKey)
		return e
	})

	if cached != "" {
		var route store.RouteData
		if jsonErr := json.Unmarshal([]byte(cached), &route); jsonErr == nil {
			return &route, nil
		}
	}

	// Fallback to PostgreSQL through circuit breaker.
	var route *store.RouteData
	dbErr := r.pgBreaker.Call(func() error {
		var e error
		if host != "" && host != r.defaultDomain {
			// White-label: find tenant by custom domain, then scope link lookup.
			tenantID, err := r.store.GetTenantIDByCustomDomain(ctx, host)
			if err != nil || tenantID == "" {
				// Unknown domain — fall back to global lookup.
				route, e = r.store.GetRouteByShortCode(ctx, shortCode)
				return e
			}
			route, e = r.store.GetRouteByShortCodeAndTenant(ctx, shortCode, tenantID)
			return e
		}
		route, e = r.store.GetRouteByShortCode(ctx, shortCode)
		return e
	})
	if dbErr != nil {
		return nil, ErrNotFound
	}

	if data, jsonErr := json.Marshal(route); jsonErr == nil {
		_ = r.redisBreaker.Call(func() error {
			return r.cache.Set(ctx, cacheKey, string(data), routeCacheTTL)
		})
	}

	return route, nil
}
