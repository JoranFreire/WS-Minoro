package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RouteData is stored in Redis cache and returned by the resolver.
type RouteData struct {
	LinkID          string        `json:"link_id"`
	TenantID        string        `json:"tenant_id"`
	ShortCode       string        `json:"short_code"`
	RoutingStrategy string        `json:"routing_strategy"`
	FallbackURL     string        `json:"fallback_url"`
	Destinations    []Destination `json:"destinations"`
}

// Destination includes Phase 2 fields for intelligent invite routing.
type Destination struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	Weight        int        `json:"weight"`
	MaxClicks     *int       `json:"max_clicks"`
	CooldownUntil *time.Time `json:"cooldown_until"`
	RiskScore     float64    `json:"risk_score"`
}

// ClickResult is returned after atomically incrementing a destination's click count.
type ClickResult struct {
	CurrentClicks int
	MaxClicks     *int
}

// ExpiredDestination is used by the health monitor to reactivate destinations.
type ExpiredDestination struct {
	ID        string
	LinkID    string
	TenantID  string
	ShortCode string
}

type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(databaseURL string) *PGStore {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		panic("failed to connect to postgres: " + err.Error())
	}
	return &PGStore{pool: pool}
}

// GetRouteByShortCode fetches active destinations including Phase 2 fields.
func (s *PGStore) GetRouteByShortCode(ctx context.Context, shortCode string) (*RouteData, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT l.id, l.tenant_id, l.short_code, l.routing_strategy, COALESCE(l.fallback_url, ''),
		       json_agg(json_build_object(
		           'id',             ld.id,
		           'url',            ld.url,
		           'weight',         ld.weight,
		           'max_clicks',     ld.max_clicks,
		           'cooldown_until', ld.cooldown_until,
		           'risk_score',     ld.risk_score
		       ))
		FROM links l
		JOIN link_destinations ld ON ld.link_id = l.id AND ld.is_active = true
		WHERE l.short_code = $1 AND l.is_active = true
		GROUP BY l.id, l.tenant_id, l.short_code, l.routing_strategy, l.fallback_url
	`, shortCode)

	var route RouteData
	var destsJSON []byte
	err := row.Scan(
		&route.LinkID, &route.TenantID, &route.ShortCode,
		&route.RoutingStrategy, &route.FallbackURL, &destsJSON,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(destsJSON, &route.Destinations); err != nil {
		return nil, err
	}

	return &route, nil
}

// IncrDestinationClicks atomically increments current_clicks and returns the result.
func (s *PGStore) IncrDestinationClicks(ctx context.Context, destID string) (ClickResult, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE link_destinations
		SET current_clicks = current_clicks + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING current_clicks, max_clicks
	`, destID)

	var result ClickResult
	err := row.Scan(&result.CurrentClicks, &result.MaxClicks)
	return result, err
}

// DisableDestination marks a destination as inactive.
func (s *PGStore) DisableDestination(ctx context.Context, destID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE link_destinations SET is_active = false, updated_at = NOW() WHERE id = $1
	`, destID)
	return err
}

// GetExpiredCooldownDestinations returns destinations whose cooldown has ended.
func (s *PGStore) GetExpiredCooldownDestinations(ctx context.Context) ([]ExpiredDestination, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ld.id, ld.link_id, l.tenant_id, l.short_code
		FROM link_destinations ld
		JOIN links l ON l.id = ld.link_id
		WHERE ld.is_active = false
		  AND ld.cooldown_until IS NOT NULL
		  AND ld.cooldown_until < NOW()
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dests []ExpiredDestination
	for rows.Next() {
		var d ExpiredDestination
		if err := rows.Scan(&d.ID, &d.LinkID, &d.TenantID, &d.ShortCode); err != nil {
			return nil, err
		}
		dests = append(dests, d)
	}
	return dests, nil
}

// ReactivateDestination re-enables a destination after its cooldown has expired.
func (s *PGStore) ReactivateDestination(ctx context.Context, destID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE link_destinations
		SET is_active = true, cooldown_until = NULL, updated_at = NOW()
		WHERE id = $1
	`, destID)
	return err
}

func (s *PGStore) Close() {
	s.pool.Close()
}

var _ = pgx.ErrNoRows
