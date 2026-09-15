package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"uuid"
)

type APIKey struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	KeyHash     string
	Label       string
	Permissions []byte
	IsActive    bool
}

type APIKeyRepository struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, key_hash, label, permissions, is_active
		FROM api_keys WHERE key_hash = $1 AND is_active = true
	`, keyHash)
	var k APIKey
	err := row.Scan(&k.ID, &k.TenantID, &k.KeyHash, &k.Label, &k.Permissions, &k.IsActive)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *APIKeyRepository) CreateAPIKey(ctx context.Context, k *APIKey) error {
	k.ID = uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO api_keys (id, tenant_id, key_hash, label, permissions, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, k.ID, k.TenantID, k.KeyHash, k.Label, k.Permissions, k.IsActive)
	return err
}

func (r *APIKeyRepository) DeleteAPIKey(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM api_keys WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	return err
}
