package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/repository"
)

type TenantService struct {
	repo *repository.Repository
}

func NewTenantService(repo *repository.Repository) *TenantService {
	return &TenantService{repo: repo}
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
	if err := s.repo.CreateAPIKey(ctx, key); err != nil {
		return "", err
	}
	return rawKey, nil
}

func (s *TenantService) DeleteAPIKey(ctx context.Context, keyID, tenantID uuid.UUID) error {
	return s.repo.DeleteAPIKey(ctx, keyID, tenantID)
}
