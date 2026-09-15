package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/repository"
)

type fakeTenantRepo struct {
	tenants   map[uuid.UUID]*repository.Tenant
	quotaByID map[uuid.UUID]int64
	quotaErr  error
	tenantErr error
}

func (f *fakeTenantRepo) GetTenantByID(ctx context.Context, id uuid.UUID) (*repository.Tenant, error) {
	if f.tenantErr != nil {
		return nil, f.tenantErr
	}
	t, ok := f.tenants[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}

func (f *fakeTenantRepo) GetQuotaUsage(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	if f.quotaErr != nil {
		return 0, f.quotaErr
	}
	return f.quotaByID[tenantID], nil
}

type fakeAPIKeyRepo struct {
	created []repository.APIKey
	deleted []uuid.UUID
	delErr  error
}

func (f *fakeAPIKeyRepo) CreateAPIKey(ctx context.Context, k *repository.APIKey) error {
	f.created = append(f.created, *k)
	return nil
}

func (f *fakeAPIKeyRepo) DeleteAPIKey(ctx context.Context, id, tenantID uuid.UUID) error {
	if f.delErr != nil {
		return f.delErr
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func TestTenantService_GetQuota_ReturnsUsedAndLimit(t *testing.T) {
	tenantID := uuid.New()
	repo := &fakeTenantRepo{
		tenants:   map[uuid.UUID]*repository.Tenant{tenantID: {ID: tenantID, QuotaClicksMonth: 50000}},
		quotaByID: map[uuid.UUID]int64{tenantID: 1234},
	}
	svc := NewTenantService(repo, &fakeAPIKeyRepo{})

	used, limit, err := svc.GetQuota(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 1234 || limit != 50000 {
		t.Fatalf("expected used=1234 limit=50000, got used=%d limit=%d", used, limit)
	}
}

func TestTenantService_GetQuota_PropagatesTenantLookupError(t *testing.T) {
	repo := &fakeTenantRepo{tenantErr: errors.New("db down")}
	svc := NewTenantService(repo, &fakeAPIKeyRepo{})

	if _, _, err := svc.GetQuota(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestTenantService_CreateAPIKey_NeverStoresRawKeyPlaintext(t *testing.T) {
	apiKeys := &fakeAPIKeyRepo{}
	svc := NewTenantService(&fakeTenantRepo{}, apiKeys)
	tenantID := uuid.New()

	rawKey, err := svc.CreateAPIKey(context.Background(), tenantID, "CI key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rawKey == "" {
		t.Fatal("expected a non-empty raw key returned to the caller")
	}
	if len(apiKeys.created) != 1 {
		t.Fatalf("expected 1 key persisted, got %d", len(apiKeys.created))
	}
	stored := apiKeys.created[0]
	if stored.KeyHash == rawKey {
		t.Fatal("the raw key must never be stored as-is — only its hash")
	}
	if stored.TenantID != tenantID || !stored.IsActive {
		t.Fatalf("unexpected stored key: %+v", stored)
	}
}

func TestTenantService_CreateAPIKey_GeneratesUniqueKeys(t *testing.T) {
	apiKeys := &fakeAPIKeyRepo{}
	svc := NewTenantService(&fakeTenantRepo{}, apiKeys)
	tenantID := uuid.New()

	keyA, err := svc.CreateAPIKey(context.Background(), tenantID, "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	keyB, err := svc.CreateAPIKey(context.Background(), tenantID, "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if keyA == keyB {
		t.Fatal("expected distinct raw keys across calls")
	}
}

func TestTenantService_DeleteAPIKey_Delegates(t *testing.T) {
	apiKeys := &fakeAPIKeyRepo{}
	svc := NewTenantService(&fakeTenantRepo{}, apiKeys)
	tenantID := uuid.New()
	keyID := uuid.New()

	if err := svc.DeleteAPIKey(context.Background(), keyID, tenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apiKeys.deleted) != 1 || apiKeys.deleted[0] != keyID {
		t.Fatalf("expected key %s to be deleted, got %+v", keyID, apiKeys.deleted)
	}
}
