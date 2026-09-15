package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"uuid"
)

type User struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	IsActive     bool
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, role, is_active
		FROM users WHERE email = $1 AND is_active = true
	`, email)
	var u User
	err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, role, is_active
		FROM users WHERE id = $1 AND is_active = true
	`, id)
	var u User
	err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
