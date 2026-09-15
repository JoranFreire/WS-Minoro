package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/repository"
)

// TenantRepo and APIKeyRepo are the subsets of their repositories
// TenantService depends on. Defined here so tests can inject fakes instead
// of requiring a live Postgres connection.
type TenantRepo interface {
	GetTenantByID(ctx context.Context, id uuid.UUID) (*repository.Tenant, error)
	GetQuotaUsage(ctx context.Context, tenantID uuid.UUID) (int64, error)
}

type APIKeyRepo interface {
	CreateAPIKey(ctx context.Context, k *repository.APIKey) error
	DeleteAPIKey(ctx context.Context, id, tenantID uuid.UUID) error
}

type TenantService struct {
	repo    TenantRepo
	apiKeys APIKeyRepo
}

func NewTenantService(repo TenantRepo, apiKeys APIKeyRepo) *TenantService {
	return &TenantService{repo: repo, apiKeys: apiKeys}
}

func (s *TenantService) GetTenant(ctx context.Context, tenantID uuid.UUID) (*repository.Tenant, error) {
	return s.repo.GetTenantByID(ctx, tenantID)
}

func (s *TenantService) GetQuota(ctx context.Context, tenantID uuid.UUID) (used, limit int64, err error) {
	tenant, err := s.repo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}
	used, err = s.repo.GetQuotaUsage(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}
	return used, tenant.QuotaClicksMonth, nil
}

func (s *TenantService) CreateAPIKey(ctx context.Context, tenantID uuid.UUID, label string) (rawKey string, err error) {
	rawKey = fmt.Sprintf("%s-%s", tenantID, uuid.New().String())
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	key := &repository.APIKey{
		TenantID:    tenantID,
		KeyHash:     keyHash,
		Label:       label,
		Permissions: []byte(`{}`),
		IsActive:    true,
	}
	if err := s.apiKeys.CreateAPIKey(ctx, key); err != nil {
		return "", err
	}
	return rawKey, nil
}

func (s *TenantService) DeleteAPIKey(ctx context.Context, keyID, tenantID uuid.UUID) error {
	return s.apiKeys.DeleteAPIKey(ctx, keyID, tenantID)
}
