package service

import (
	"context"
	"errors"

	"github.com/ws-minoro/link-admin/internal/billing"
	"github.com/ws-minoro/link-admin/internal/repository"
	"uuid"
)

var ErrBillingNotConfigured = errors.New("billing not configured")
var ErrUnknownPlan = errors.New("unknown plan")

const (
	FreePlan            = "free"
	freePlanQuotaClicks = 10_000 // matches the tenants table DEFAULT
)

// PlanQuotas maps a plan name to its monthly click quota — the actual
// metered resource in the monetization model. Pricing itself lives in
// Pagar.me's dashboard (referenced by plan ID via config), never here.
var PlanQuotas = map[string]int64{
	"starter":  50_000,
	"pro":      500_000,
	"business": 5_000_000,
}

// PagarmeClient is the subset of billing.Client BillingService depends on.
// Defined here so tests can inject a fake instead of calling the real
// Pagar.me API.
type PagarmeClient interface {
	CreateSubscription(ctx context.Context, in billing.CreateSubscriptionInput) (*billing.Subscription, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
}

// BillingTenantRepo is the subset of repository.TenantRepository
// BillingService depends on.
type BillingTenantRepo interface {
	GetTenantByID(ctx context.Context, id uuid.UUID) (*repository.Tenant, error)
	GetTenantByPagarmeCustomerID(ctx context.Context, customerID string) (*repository.Tenant, error)
	SetPagarmeCustomerID(ctx context.Context, tenantID uuid.UUID, customerID string) error
	UpdateSubscription(ctx context.Context, tenantID uuid.UUID, plan string, quotaClicksMonth int64, subscriptionID string) error
}

// QuotaUnblocker clears the over-quota flag link-router checks on the
// redirect hot path, so an upgrade takes effect immediately instead of
// waiting for the flag to expire at month's end.
type QuotaUnblocker interface {
	Clear(ctx context.Context, tenantID string) error
}

type BillingService struct {
	tenants BillingTenantRepo
	users   UserGetter
	client  PagarmeClient
	unblock QuotaUnblocker
	planIDs map[string]string
}

// NewBillingService builds the service. A nil client means billing isn't
// configured (e.g. PAGARME_SECRET_KEY unset) — every operation then returns
// ErrBillingNotConfigured instead of the service failing to start, so a
// deployment that doesn't use billing yet is unaffected.
func NewBillingService(tenants BillingTenantRepo, users UserGetter, client PagarmeClient, unblock QuotaUnblocker, planIDs map[string]string) *BillingService {
	return &BillingService{tenants: tenants, users: users, client: client, unblock: unblock, planIDs: planIDs}
}

func (s *BillingService) Configured() bool {
	return s.client != nil
}

// Subscribe charges the tokenized card for the given plan and upgrades the
// tenant on success. cardToken comes from Pagar.me's client-side
// tokenizecard.js — the raw card number never reaches this service.
func (s *BillingService) Subscribe(ctx context.Context, tenantID, userID uuid.UUID, planName, cardToken string) (quotaClicksMonth int64, err error) {
	if !s.Configured() {
		return 0, ErrBillingNotConfigured
	}

	planID, ok := s.planIDs[planName]
	if !ok {
		return 0, ErrUnknownPlan
	}
	quota, ok := PlanQuotas[planName]
	if !ok {
		return 0, ErrUnknownPlan
	}

	tenant, err := s.tenants.GetTenantByID(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return 0, err
	}

	sub, err := s.client.CreateSubscription(ctx, billing.CreateSubscriptionInput{
		PlanID:        planID,
		CustomerName:  tenant.Name,
		CustomerEmail: user.Email,
		CardToken:     cardToken,
	})
	if err != nil {
		return 0, err
	}

	if err := s.tenants.UpdateSubscription(ctx, tenantID, planName, quota, sub.ID); err != nil {
		return 0, err
	}
	if sub.Customer.ID != "" {
		_ = s.tenants.SetPagarmeCustomerID(ctx, tenantID, sub.Customer.ID)
	}
	_ = s.unblock.Clear(ctx, tenantID.String())

	return quota, nil
}

// HandleChargePaid reacts to a Pagar.me `charge.paid` webhook by clearing
// any pending quota block for the tenant that owns the paying customer.
func (s *BillingService) HandleChargePaid(ctx context.Context, pagarmeCustomerID string) error {
	tenant, err := s.tenants.GetTenantByPagarmeCustomerID(ctx, pagarmeCustomerID)
	if err != nil {
		return err
	}
	return s.unblock.Clear(ctx, tenant.ID.String())
}

// HandleSubscriptionCanceled reacts to a Pagar.me `subscription.canceled`
// webhook by reverting the tenant to the free plan.
func (s *BillingService) HandleSubscriptionCanceled(ctx context.Context, pagarmeCustomerID string) error {
	tenant, err := s.tenants.GetTenantByPagarmeCustomerID(ctx, pagarmeCustomerID)
	if err != nil {
		return err
	}
	return s.tenants.UpdateSubscription(ctx, tenant.ID, FreePlan, freePlanQuotaClicks, "")
}
