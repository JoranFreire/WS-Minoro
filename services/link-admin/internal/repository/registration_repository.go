package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmailTaken is returned when the email is already registered to a user.
var ErrEmailTaken = errors.New("email already registered")

type RegistrationRepository struct {
	pool *pgxpool.Pool
}

func NewRegistrationRepository(pool *pgxpool.Pool) *RegistrationRepository {
	return &RegistrationRepository{pool: pool}
}

// RegisterTenantOwner creates a new tenant and its first user (role "owner")
// in a single transaction — self-serve signup must never leave an orphaned
// tenant behind if user creation fails (e.g. duplicate email).
func (r *RegistrationRepository) RegisterTenantOwner(ctx context.Context, tenantName, email, passwordHash string) (*Tenant, *User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var tenant Tenant
	err = tx.QueryRow(ctx, `
		INSERT INTO tenants (name)
		VALUES ($1)
		RETURNING id, name, plan, quota_clicks_month, COALESCE(custom_domain, ''), is_active
	`, tenantName).Scan(&tenant.ID, &tenant.Name, &tenant.Plan, &tenant.QuotaClicksMonth, &tenant.CustomDomain, &tenant.IsActive)
	if err != nil {
		return nil, nil, err
	}

	var user User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, role)
		VALUES ($1, $2, $3, 'owner')
		RETURNING id, tenant_id, email, password_hash, role, is_active
	`, tenant.ID, email, passwordHash).Scan(&user.ID, &user.TenantID, &user.Email, &user.PasswordHash, &user.Role, &user.IsActive)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, nil, ErrEmailTaken
		}
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return &tenant, &user, nil
}
