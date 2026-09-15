package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"uuid"
)

type Link struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	ShortCode       string    `json:"short_code"`
	Title           string    `json:"title"`
	FallbackURL     string    `json:"fallback_url"`
	RoutingStrategy string    `json:"routing_strategy"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Destination struct {
	ID            uuid.UUID `json:"id"`
	LinkID        uuid.UUID `json:"link_id"`
	URL           string    `json:"url"`
	Weight        int       `json:"weight"`
	MaxClicks     *int      `json:"max_clicks"`
	CurrentClicks int       `json:"current_clicks"`
	IsActive      bool      `json:"is_active"`
}

type LinkRepository struct {
	pool *pgxpool.Pool
}

func NewLinkRepository(pool *pgxpool.Pool) *LinkRepository {
	return &LinkRepository{pool: pool}
}

func (r *LinkRepository) ListLinks(ctx context.Context, tenantID uuid.UUID) ([]Link, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, short_code, COALESCE(title,''), COALESCE(fallback_url,''),
		       routing_strategy, is_active, created_at, updated_at
		FROM links WHERE tenant_id = $1 ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.TenantID, &l.ShortCode, &l.Title, &l.FallbackURL,
			&l.RoutingStrategy, &l.IsActive, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

func (r *LinkRepository) GetLinkByID(ctx context.Context, id, tenantID uuid.UUID) (*Link, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, short_code, COALESCE(title,''), COALESCE(fallback_url,''),
		       routing_strategy, is_active, created_at, updated_at
		FROM links WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)
	var l Link
	err := row.Scan(&l.ID, &l.TenantID, &l.ShortCode, &l.Title, &l.FallbackURL,
		&l.RoutingStrategy, &l.IsActive, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *LinkRepository) CreateLink(ctx context.Context, l *Link) error {
	l.ID = uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO links (id, tenant_id, short_code, title, fallback_url, routing_strategy, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, l.ID, l.TenantID, l.ShortCode, l.Title, l.FallbackURL, l.RoutingStrategy, l.IsActive)
	return err
}

func (r *LinkRepository) UpdateLink(ctx context.Context, l *Link) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE links SET title=$1, fallback_url=$2, routing_strategy=$3, is_active=$4, updated_at=NOW()
		WHERE id=$5 AND tenant_id=$6
	`, l.Title, l.FallbackURL, l.RoutingStrategy, l.IsActive, l.ID, l.TenantID)
	return err
}

func (r *LinkRepository) DeleteLink(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM links WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	return err
}

func (r *LinkRepository) ListDestinations(ctx context.Context, linkID uuid.UUID) ([]Destination, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, link_id, url, weight, max_clicks, current_clicks, is_active
		FROM link_destinations WHERE link_id = $1
	`, linkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dests []Destination
	for rows.Next() {
		var d Destination
		if err := rows.Scan(&d.ID, &d.LinkID, &d.URL, &d.Weight, &d.MaxClicks, &d.CurrentClicks, &d.IsActive); err != nil {
			return nil, err
		}
		dests = append(dests, d)
	}
	return dests, nil
}

func (r *LinkRepository) CreateDestination(ctx context.Context, d *Destination) error {
	d.ID = uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO link_destinations (id, link_id, url, weight, max_clicks, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, d.ID, d.LinkID, d.URL, d.Weight, d.MaxClicks, d.IsActive)
	return err
}

func (r *LinkRepository) UpdateDestination(ctx context.Context, d *Destination) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE link_destinations SET url=$1, weight=$2, max_clicks=$3, is_active=$4, updated_at=NOW()
		WHERE id=$5 AND link_id=$6
	`, d.URL, d.Weight, d.MaxClicks, d.IsActive, d.ID, d.LinkID)
	return err
}

func (r *LinkRepository) DeleteDestination(ctx context.Context, destID, linkID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM link_destinations WHERE id=$1 AND link_id=$2`, destID, linkID)
	return err
}
