package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"uuid"
)

type Tenant struct {
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	Plan                  string    `json:"plan"`
	QuotaClicksMonth      int64     `json:"quota_clicks_month"`
	CustomDomain          string    `json:"custom_domain"`
	IsActive              bool      `json:"is_active"`
	PagarmeCustomerID     string    `json:"-"`
	PagarmeSubscriptionID string    `json:"-"`
}

type TenantRepository struct {
	pool *pgxpool.Pool
}

func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

const tenantColumns = `id, name, plan, quota_clicks_month, COALESCE(custom_domain,''), is_active,
	COALESCE(pagarme_customer_id,''), COALESCE(pagarme_subscription_id,'')`

func scanTenant(row pgx.Row) (*Tenant, error) {
	var t Tenant
	err := row.Scan(&t.ID, &t.Name, &t.Plan, &t.QuotaClicksMonth, &t.CustomDomain, &t.IsActive,
		&t.PagarmeCustomerID, &t.PagarmeSubscriptionID)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TenantRepository) GetTenantByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+tenantColumns+` FROM tenants WHERE id = $1`, id)
	return scanTenant(row)
}

// GetTenantByPagarmeCustomerID looks up the tenant a Pagar.me webhook event
// belongs to.
func (r *TenantRepository) GetTenantByPagarmeCustomerID(ctx context.Context, customerID string) (*Tenant, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+tenantColumns+` FROM tenants WHERE pagarme_customer_id = $1`, customerID)
	return scanTenant(row)
}

func (r *TenantRepository) SetPagarmeCustomerID(ctx context.Context, tenantID uuid.UUID, customerID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE tenants SET pagarme_customer_id = $1 WHERE id = $2`, customerID, tenantID)
	return err
}

// UpdateSubscription applies a plan change confirmed by Pagar.me (via the
// checkout flow or a webhook) — the new plan name, its click quota, and the
// Pagar.me subscription id backing it.
func (r *TenantRepository) UpdateSubscription(ctx context.Context, tenantID uuid.UUID, plan string, quotaClicksMonth int64, subscriptionID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tenants SET plan = $1, quota_clicks_month = $2, pagarme_subscription_id = $3
		WHERE id = $4
	`, plan, quotaClicksMonth, subscriptionID, tenantID)
	return err
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
