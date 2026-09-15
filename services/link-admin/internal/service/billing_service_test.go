package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/billing"
	"github.com/ws-minoro/link-admin/internal/repository"
)

type fakeBillingTenantRepo struct {
	tenants            map[uuid.UUID]*repository.Tenant
	byCustomerID       map[string]*repository.Tenant
	setCustomerIDCalls []struct {
		tenantID   uuid.UUID
		customerID string
	}
	updateCalls []struct {
		tenantID       uuid.UUID
		plan           string
		quota          int64
		subscriptionID string
	}
}

func newFakeBillingTenantRepo() *fakeBillingTenantRepo {
	return &fakeBillingTenantRepo{
		tenants:      map[uuid.UUID]*repository.Tenant{},
		byCustomerID: map[string]*repository.Tenant{},
	}
}

func (f *fakeBillingTenantRepo) GetTenantByID(ctx context.Context, id uuid.UUID) (*repository.Tenant, error) {
	t, ok := f.tenants[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}

func (f *fakeBillingTenantRepo) GetTenantByPagarmeCustomerID(ctx context.Context, customerID string) (*repository.Tenant, error) {
	t, ok := f.byCustomerID[customerID]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}

func (f *fakeBillingTenantRepo) SetPagarmeCustomerID(ctx context.Context, tenantID uuid.UUID, customerID string) error {
	f.setCustomerIDCalls = append(f.setCustomerIDCalls, struct {
		tenantID   uuid.UUID
		customerID string
	}{tenantID, customerID})
	return nil
}

func (f *fakeBillingTenantRepo) UpdateSubscription(ctx context.Context, tenantID uuid.UUID, plan string, quotaClicksMonth int64, subscriptionID string) error {
	f.updateCalls = append(f.updateCalls, struct {
		tenantID       uuid.UUID
		plan           string
		quota          int64
		subscriptionID string
	}{tenantID, plan, quotaClicksMonth, subscriptionID})
	return nil
}

type fakePagarmeClient struct {
	sub        *billing.Subscription
	createErr  error
	createdFor []billing.CreateSubscriptionInput
}

func (f *fakePagarmeClient) CreateSubscription(ctx context.Context, in billing.CreateSubscriptionInput) (*billing.Subscription, error) {
	f.createdFor = append(f.createdFor, in)
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.sub, nil
}

func (f *fakePagarmeClient) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return nil
}

type fakeQuotaUnblocker struct {
	cleared []string
}

func (f *fakeQuotaUnblocker) Clear(ctx context.Context, tenantID string) error {
	f.cleared = append(f.cleared, tenantID)
	return nil
}

func newTestBillingService(t *testing.T, configured bool) (*BillingService, *fakeBillingTenantRepo, *fakeUserGetter, *fakePagarmeClient, *fakeQuotaUnblocker) {
	t.Helper()
	tenants := newFakeBillingTenantRepo()
	users := &fakeUserGetter{usersByEmail: map[string]*repository.User{}}
	unblock := &fakeQuotaUnblocker{}

	var client PagarmeClient
	fake := &fakePagarmeClient{sub: &billing.Subscription{ID: "sub_123"}}
	fake.sub.Customer.ID = "cus_123"
	if configured {
		client = fake
	}

	svc := NewBillingService(tenants, users, client, unblock, map[string]string{
		"starter": "plan_starter_id",
	})
	return svc, tenants, users, fake, unblock
}

func seedUser(users *fakeUserGetter, id uuid.UUID, email string) {
	users.usersByEmail[email] = &repository.User{ID: id, Email: email}
}

func TestBillingService_Configured(t *testing.T) {
	svc, _, _, _, _ := newTestBillingService(t, true)
	if !svc.Configured() {
		t.Fatal("expected configured service to report Configured() true")
	}

	svc2, _, _, _, _ := newTestBillingService(t, false)
	if svc2.Configured() {
		t.Fatal("expected unconfigured service to report Configured() false")
	}
}

func TestSubscribe_NotConfiguredReturnsError(t *testing.T) {
	svc, _, _, _, _ := newTestBillingService(t, false)

	_, err := svc.Subscribe(context.Background(), uuid.New(), uuid.New(), "starter", "token_abc")
	if !errors.Is(err, ErrBillingNotConfigured) {
		t.Fatalf("expected ErrBillingNotConfigured, got %v", err)
	}
}

func TestSubscribe_UnknownPlanRejected(t *testing.T) {
	svc, _, _, _, _ := newTestBillingService(t, true)

	_, err := svc.Subscribe(context.Background(), uuid.New(), uuid.New(), "enterprise-deluxe", "token_abc")
	if !errors.Is(err, ErrUnknownPlan) {
		t.Fatalf("expected ErrUnknownPlan, got %v", err)
	}
}

func TestSubscribe_Success(t *testing.T) {
	svc, tenants, users, client, unblock := newTestBillingService(t, true)
	tenantID := uuid.New()
	userID := uuid.New()
	tenants.tenants[tenantID] = &repository.Tenant{ID: tenantID, Name: "Acme Inc"}
	seedUser(users, userID, "owner@acme.com")

	quota, err := svc.Subscribe(context.Background(), tenantID, userID, "starter", "token_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quota != PlanQuotas["starter"] {
		t.Fatalf("expected quota %d, got %d", PlanQuotas["starter"], quota)
	}

	if len(client.createdFor) != 1 {
		t.Fatalf("expected 1 subscription creation call, got %d", len(client.createdFor))
	}
	if client.createdFor[0].CardToken != "token_abc" || client.createdFor[0].PlanID != "plan_starter_id" {
		t.Fatalf("unexpected subscription input: %+v", client.createdFor[0])
	}

	if len(tenants.updateCalls) != 1 || tenants.updateCalls[0].plan != "starter" || tenants.updateCalls[0].subscriptionID != "sub_123" {
		t.Fatalf("unexpected update calls: %+v", tenants.updateCalls)
	}
	if len(tenants.setCustomerIDCalls) != 1 || tenants.setCustomerIDCalls[0].customerID != "cus_123" {
		t.Fatalf("unexpected customer id calls: %+v", tenants.setCustomerIDCalls)
	}
	if len(unblock.cleared) != 1 || unblock.cleared[0] != tenantID.String() {
		t.Fatalf("expected the tenant's quota block cleared immediately, got %+v", unblock.cleared)
	}
}

func TestSubscribe_PagarmeErrorPropagatesAndSkipsUpdate(t *testing.T) {
	svc, tenants, users, client, unblock := newTestBillingService(t, true)
	client.createErr = errors.New("card declined")
	tenantID := uuid.New()
	userID := uuid.New()
	tenants.tenants[tenantID] = &repository.Tenant{ID: tenantID, Name: "Acme Inc"}
	seedUser(users, userID, "owner@acme.com")

	_, err := svc.Subscribe(context.Background(), tenantID, userID, "starter", "token_abc")
	if err == nil {
		t.Fatal("expected an error when the card is declined")
	}
	if len(tenants.updateCalls) != 0 {
		t.Fatal("must not update the tenant's plan when the charge fails")
	}
	if len(unblock.cleared) != 0 {
		t.Fatal("must not clear the quota block when the charge fails")
	}
}

func TestHandleChargePaid_ClearsQuotaBlockForOwningTenant(t *testing.T) {
	svc, tenants, _, _, unblock := newTestBillingService(t, true)
	tenantID := uuid.New()
	tenants.byCustomerID["cus_123"] = &repository.Tenant{ID: tenantID}

	if err := svc.HandleChargePaid(context.Background(), "cus_123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(unblock.cleared) != 1 || unblock.cleared[0] != tenantID.String() {
		t.Fatalf("expected tenant %s unblocked, got %+v", tenantID, unblock.cleared)
	}
}

func TestHandleChargePaid_UnknownCustomerReturnsError(t *testing.T) {
	svc, _, _, _, unblock := newTestBillingService(t, true)

	if err := svc.HandleChargePaid(context.Background(), "cus_unknown"); err == nil {
		t.Fatal("expected an error for an unknown customer id")
	}
	if len(unblock.cleared) != 0 {
		t.Fatal("must not clear anything for an unknown customer")
	}
}

func TestHandleSubscriptionCanceled_RevertsToFreePlan(t *testing.T) {
	svc, tenants, _, _, _ := newTestBillingService(t, true)
	tenantID := uuid.New()
	tenants.byCustomerID["cus_123"] = &repository.Tenant{ID: tenantID}

	if err := svc.HandleSubscriptionCanceled(context.Background(), "cus_123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(tenants.updateCalls))
	}
	got := tenants.updateCalls[0]
	if got.plan != FreePlan || got.quota != freePlanQuotaClicks {
		t.Fatalf("expected revert to free plan, got %+v", got)
	}
}
