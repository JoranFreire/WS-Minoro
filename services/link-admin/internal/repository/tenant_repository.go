package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Tenant struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Plan             string    `json:"plan"`
	QuotaClicksMonth int64     `json:"quota_clicks_month"`
	CustomDomain     string    `json:"custom_domain"`
	IsActive         bool      `json:"is_active"`
}

type TenantRepository struct {
	pool *pgxpool.Pool
}

func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

func (r *TenantRepository) GetTenantByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, plan, quota_clicks_month, COALESCE(custom_domain,''), is_active
		FROM tenants WHERE id = $1
	`, id)
	var t Tenant
	err := row.Scan(&t.ID, &t.Name, &t.Plan, &t.QuotaClicksMonth, &t.CustomDomain, &t.IsActive)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepository) GetQuotaUsage(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var used int64
	month := time.Now().Format("2006-01-02")
	row := r.pool.QueryRow(ctx, `
		SELECT COALESCE(clicks_used, 0) FROM quota_usage
		WHERE tenant_id = $1 AND month = date_trunc('month', $2::date)
	`, tenantID, month)
	_ = row.Scan(&used)
	return used, nil
}
