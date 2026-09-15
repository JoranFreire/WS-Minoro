package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/ws-minoro/link-router/internal/breaker"
	"github.com/ws-minoro/link-router/internal/cache"
	"github.com/ws-minoro/link-router/internal/event"
	"github.com/ws-minoro/link-router/internal/store"
)

// fakeStore is an in-memory double for the Store interface.
type fakeStore struct {
	mu sync.Mutex

	routesByCode    map[string]*store.RouteData
	tenantByDomain  map[string]string
	clicks          map[string]store.ClickResult
	maxClicksByDest map[string]*int
	disabled        map[string]bool

	getRouteErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		routesByCode:    map[string]*store.RouteData{},
		tenantByDomain:  map[string]string{},
		clicks:          map[string]store.ClickResult{},
		maxClicksByDest: map[string]*int{},
		disabled:        map[string]bool{},
	}
}

func (f *fakeStore) GetRouteByShortCode(ctx context.Context, shortCode string) (*store.RouteData, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getRouteErr != nil {
		return nil, f.getRouteErr
	}
	route, ok := f.routesByCode[shortCode]
	if !ok {
		return nil, errors.New("not found")
	}
	return route, nil
}

func (f *fakeStore) GetRouteByShortCodeAndTenant(ctx context.Context, shortCode, tenantID string) (*store.RouteData, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	route, ok := f.routesByCode[shortCode]
	if !ok || route.TenantID != tenantID {
		return nil, errors.New("not found")
	}
	return route, nil
}

func (f *fakeStore) GetTenantIDByCustomDomain(ctx context.Context, domain string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tenantID, ok := f.tenantByDomain[domain]
	if !ok {
		return "", errors.New("not found")
	}
	return tenantID, nil
}

func (f *fakeStore) IncrDestinationClicks(ctx context.Context, destID string) (store.ClickResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := f.clicks[destID]
	result.CurrentClicks++
	result.MaxClicks = f.maxClicksByDest[destID]
	f.clicks[destID] = result
	return result, nil
}

func (f *fakeStore) DisableDestination(ctx context.Context, destID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.disabled[destID] = true
	return nil
}

// fakeHealthPublisher is an in-memory double for the HealthPublisher interface.
type fakeHealthPublisher struct {
	mu        sync.Mutex
	published []event.HealthEvent
}

func (f *fakeHealthPublisher) PublishDisabled(ctx context.Context, evt event.HealthEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published = append(f.published, evt)
}

func (f *fakeHealthPublisher) events() []event.HealthEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]event.HealthEvent, len(f.published))
	copy(out, f.published)
	return out
}

func newTestResolver(t *testing.T, fs *fakeStore, fhp *fakeHealthPublisher, maxRiskScore float64) *LinkResolver {
	t.Helper()
	mr := miniredis.RunT(t)
	c := cache.NewRedisCacheWithClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}))

	return &LinkResolver{
		cache:         c,
		store:         fs,
		healthPub:     fhp,
		maxRiskScore:  maxRiskScore,
		defaultDomain: "default.test",
		redisBreaker:  breaker.New(1000, time.Hour),
		pgBreaker:     breaker.New(1000, time.Hour),
	}
}

func TestResolve_FallsBackToStoreOnCacheMiss(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID:          "link-1",
		TenantID:        "tenant-1",
		ShortCode:       "abc",
		RoutingStrategy: "single",
		Destinations: []store.Destination{
			{ID: "d1", URL: "https://example.com/a", Weight: 1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)

	url, linkID, tenantID, _, err := r.Resolve(context.Background(), "abc", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com/a" || linkID != "link-1" || tenantID != "tenant-1" {
		t.Fatalf("unexpected result: url=%s linkID=%s tenantID=%s", url, linkID, tenantID)
	}
}

func TestResolve_UsesCacheOnSecondCall(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID:    "link-1",
		TenantID:  "tenant-1",
		ShortCode: "abc",
		Destinations: []store.Destination{
			{ID: "d1", URL: "https://example.com/a", Weight: 1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)
	ctx := context.Background()

	if _, _, _, _, err := r.Resolve(ctx, "abc", "", ""); err != nil {
		t.Fatalf("first resolve: %v", err)
	}

	// Break the store after warming the cache — a second resolve must not need it.
	fs.getRouteErr = errors.New("store should not be hit")

	url, _, _, _, err := r.Resolve(ctx, "abc", "", "")
	if err != nil {
		t.Fatalf("second resolve should hit cache, got error: %v", err)
	}
	if url != "https://example.com/a" {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestResolve_UnknownShortCodeReturnsErrNotFound(t *testing.T) {
	fs := newFakeStore()
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)

	_, _, _, _, err := r.Resolve(context.Background(), "missing", "", "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResolve_FiltersOutRiskyDestinations(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "t1", ShortCode: "abc",
		Destinations: []store.Destination{
			{ID: "risky", URL: "https://risky.example.com", Weight: 1, RiskScore: 0.9},
			{ID: "safe", URL: "https://safe.example.com", Weight: 1, RiskScore: 0.1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)

	url, _, _, _, err := r.Resolve(context.Background(), "abc", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://safe.example.com" {
		t.Fatalf("expected risky destination to be filtered out, got %s", url)
	}
}

func TestResolve_FiltersOutCooldownDestinations(t *testing.T) {
	future := time.Now().Add(time.Hour)
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "t1", ShortCode: "abc",
		Destinations: []store.Destination{
			{ID: "cooling", URL: "https://cooling.example.com", Weight: 1, CooldownUntil: &future},
			{ID: "safe", URL: "https://safe.example.com", Weight: 1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)

	url, _, _, _, err := r.Resolve(context.Background(), "abc", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://safe.example.com" {
		t.Fatalf("expected cooldown destination to be filtered out, got %s", url)
	}
}

func TestResolve_NoActiveDestinationsUsesFallback(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "t1", ShortCode: "abc",
		FallbackURL: "https://fallback.example.com",
		Destinations: []store.Destination{
			{ID: "risky", URL: "https://risky.example.com", Weight: 1, RiskScore: 0.9},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)

	url, linkID, tenantID, _, err := r.Resolve(context.Background(), "abc", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://fallback.example.com" || linkID != "link-1" || tenantID != "t1" {
		t.Fatalf("unexpected fallback result: url=%s linkID=%s tenantID=%s", url, linkID, tenantID)
	}
}

func TestResolve_NoActiveDestinationsNoFallbackReturnsNotFound(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "t1", ShortCode: "abc",
		Destinations: []store.Destination{
			{ID: "risky", URL: "https://risky.example.com", Weight: 1, RiskScore: 0.9},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)

	_, _, _, _, err := r.Resolve(context.Background(), "abc", "", "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResolve_WeightedStrategyOnlyPicksActiveDestinations(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "t1", ShortCode: "abc",
		RoutingStrategy: "weighted",
		Destinations: []store.Destination{
			{ID: "a", URL: "https://a.example.com", Weight: 1},
			{ID: "b", URL: "https://b.example.com", Weight: 1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)

	valid := map[string]bool{"https://a.example.com": true, "https://b.example.com": true}
	for range 20 {
		url, _, _, _, err := r.Resolve(context.Background(), "abc", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !valid[url] {
			t.Fatalf("unexpected url selected: %s", url)
		}
	}
}

func TestResolve_CustomDomainScopesLookupToTenant(t *testing.T) {
	fs := newFakeStore()
	fs.tenantByDomain["brand.example.com"] = "tenant-42"
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "tenant-42", ShortCode: "abc",
		Destinations: []store.Destination{
			{ID: "d1", URL: "https://scoped.example.com", Weight: 1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)

	url, _, tenantID, _, err := r.Resolve(context.Background(), "abc", "brand.example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://scoped.example.com" || tenantID != "tenant-42" {
		t.Fatalf("unexpected result: url=%s tenantID=%s", url, tenantID)
	}
}

func TestResolve_UnknownCustomDomainFallsBackToGlobalLookup(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "t1", ShortCode: "abc",
		Destinations: []store.Destination{
			{ID: "d1", URL: "https://global.example.com", Weight: 1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)

	url, _, _, _, err := r.Resolve(context.Background(), "abc", "unknown-domain.example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://global.example.com" {
		t.Fatalf("expected fallback to global lookup, got %s", url)
	}
}

func TestTrackClick_AutoDisablesAndPublishesWhenMaxClicksReached(t *testing.T) {
	fs := newFakeStore()
	maxClicks := 1
	fs.maxClicksByDest["d1"] = &maxClicks
	fhp := &fakeHealthPublisher{}
	r := newTestResolver(t, fs, fhp, 0.7)

	route := &store.RouteData{LinkID: "link-1", TenantID: "t1", ShortCode: "abc"}
	dest := store.Destination{ID: "d1", MaxClicks: &maxClicks}

	r.trackClick(route, dest)

	if !fs.disabled["d1"] {
		t.Fatal("expected destination to be disabled after reaching max_clicks")
	}
	events := fhp.events()
	if len(events) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(events))
	}
	if events[0].DestinationID != "d1" || events[0].Reason != "max_clicks_reached" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

func TestTrackClick_DoesNotDisableBelowMaxClicks(t *testing.T) {
	fs := newFakeStore()
	maxClicks := 5
	fs.maxClicksByDest["d1"] = &maxClicks
	fhp := &fakeHealthPublisher{}
	r := newTestResolver(t, fs, fhp, 0.7)

	route := &store.RouteData{LinkID: "link-1", TenantID: "t1", ShortCode: "abc"}
	dest := store.Destination{ID: "d1", MaxClicks: &maxClicks}

	r.trackClick(route, dest)

	if fs.disabled["d1"] {
		t.Fatal("destination should not be disabled before reaching max_clicks")
	}
	if len(fhp.events()) != 0 {
		t.Fatal("no event should be published before reaching max_clicks")
	}
}

func TestTrackClick_NoMaxClicksNeverDisables(t *testing.T) {
	fs := newFakeStore()
	fhp := &fakeHealthPublisher{}
	r := newTestResolver(t, fs, fhp, 0.7)

	route := &store.RouteData{LinkID: "link-1", TenantID: "t1", ShortCode: "abc"}
	dest := store.Destination{ID: "d1", MaxClicks: nil}

	for range 10 {
		r.trackClick(route, dest)
	}

	if fs.disabled["d1"] {
		t.Fatal("destination without max_clicks should never be disabled")
	}
}

// Ensure the cache we pre-seed round-trips through JSON exactly the way
// getRoute expects, guarding against silent (de)serialization drift.
func TestRouteData_JSONRoundTrip(t *testing.T) {
	original := store.RouteData{
		LinkID: "link-1", TenantID: "t1", ShortCode: "abc",
		RoutingStrategy: "weighted",
		Destinations: []store.Destination{
			{ID: "d1", URL: "https://example.com", Weight: 3},
		},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded store.RouteData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.LinkID != original.LinkID || len(decoded.Destinations) != 1 {
		t.Fatalf("round trip mismatch: %+v", decoded)
	}
}

func TestResolve_OverQuotaTenantGetsFallbackURL(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "tenant-over-quota", ShortCode: "abc",
		FallbackURL: "https://fallback.example.com",
		Destinations: []store.Destination{
			{ID: "d1", URL: "https://example.com/a", Weight: 1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)
	ctx := context.Background()
	if err := r.cache.Set(ctx, quotaBlockedKey("tenant-over-quota"), "1", time.Hour); err != nil {
		t.Fatalf("failed to seed quota flag: %v", err)
	}

	url, _, _, _, err := r.Resolve(ctx, "abc", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://fallback.example.com" {
		t.Fatalf("expected fallback url for over-quota tenant, got %s", url)
	}
}

func TestResolve_OverQuotaTenantWithNoFallbackReturnsNotFound(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "tenant-over-quota", ShortCode: "abc",
		Destinations: []store.Destination{
			{ID: "d1", URL: "https://example.com/a", Weight: 1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)
	ctx := context.Background()
	if err := r.cache.Set(ctx, quotaBlockedKey("tenant-over-quota"), "1", time.Hour); err != nil {
		t.Fatalf("failed to seed quota flag: %v", err)
	}

	_, _, _, _, err := r.Resolve(ctx, "abc", "", "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResolve_OverQuotaTenantNeverTracksAClick(t *testing.T) {
	fs := newFakeStore()
	maxClicks := 100
	fs.maxClicksByDest["d1"] = &maxClicks
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "tenant-over-quota", ShortCode: "abc",
		Destinations: []store.Destination{
			{ID: "d1", URL: "https://example.com/a", Weight: 1, MaxClicks: &maxClicks},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)
	ctx := context.Background()
	if err := r.cache.Set(ctx, quotaBlockedKey("tenant-over-quota"), "1", time.Hour); err != nil {
		t.Fatalf("failed to seed quota flag: %v", err)
	}

	if _, _, _, _, err := r.Resolve(ctx, "abc", "", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// give the (nonexistent) async trackClick goroutine a chance to run,
	// to make sure it really was never launched.
	time.Sleep(20 * time.Millisecond)
	if fs.clicks["d1"].CurrentClicks != 0 {
		t.Fatalf("expected no click tracked for an over-quota tenant, got %+v", fs.clicks["d1"])
	}
}

func TestResolve_UnderQuotaTenantUnaffected(t *testing.T) {
	fs := newFakeStore()
	fs.routesByCode["abc"] = &store.RouteData{
		LinkID: "link-1", TenantID: "tenant-fine", ShortCode: "abc",
		Destinations: []store.Destination{
			{ID: "d1", URL: "https://example.com/a", Weight: 1},
		},
	}
	r := newTestResolver(t, fs, &fakeHealthPublisher{}, 0.7)
	// No quota flag seeded for "tenant-fine".

	url, _, _, _, err := r.Resolve(context.Background(), "abc", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com/a" {
		t.Fatalf("expected normal resolution, got %s", url)
	}
}
