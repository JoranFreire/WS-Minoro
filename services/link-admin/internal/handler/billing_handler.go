package handler

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/middleware"
	"github.com/ws-minoro/link-admin/internal/service"
)

type BillingHandler struct {
	billingSvc  *service.BillingService
	webhookUser string
	webhookPass string
}

func NewBillingHandler(billingSvc *service.BillingService, webhookUser, webhookPass string) *BillingHandler {
	return &BillingHandler{billingSvc: billingSvc, webhookUser: webhookUser, webhookPass: webhookPass}
}

func getUserID(c *fiber.Ctx) (uuid.UUID, error) {
	claims, ok := c.Locals(middleware.UserContextKey).(*service.Claims)
	if !ok {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	return uuid.Parse(claims.UserID)
}

// Subscribe handles POST /api/v1/billing/subscribe. cardToken must already
// be a single-use Pagar.me card_token produced client-side by
// tokenizecard.js — never a raw card number.
func (h *BillingHandler) Subscribe(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req struct {
		Plan      string `json:"plan"`
		CardToken string `json:"card_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	quota, err := h.billingSvc.Subscribe(c.Context(), tenantID, userID, req.Plan, req.CardToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBillingNotConfigured):
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "billing is not configured"})
		case errors.Is(err, service.ErrUnknownPlan):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unknown plan"})
		default:
			log.Printf("billing: subscribe tenant=%s plan=%s: %v", tenantID, req.Plan, err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to create subscription"})
		}
	}

	return c.JSON(fiber.Map{"plan": req.Plan, "quota_clicks_month": quota})
}

type pagarmeWebhookPayload struct {
	Type string `json:"type"`
	Data struct {
		Customer struct {
			ID string `json:"id"`
		} `json:"customer"`
	} `json:"data"`
}

// Webhook handles POST /webhooks/pagarme. Authenticated via Basic Auth
// credentials set in the Pagar.me dashboard when the webhook endpoint is
// registered (PAGARME_WEBHOOK_USER / PAGARME_WEBHOOK_PASSWORD here) — not a
// JWT, since Pagar.me is the caller, not a logged-in user.
func (h *BillingHandler) Webhook(c *fiber.Ctx) error {
	if !h.verifyWebhookAuth(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var payload pagarmeWebhookPayload
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	var handleErr error
	switch payload.Type {
	case "charge.paid":
		handleErr = h.billingSvc.HandleChargePaid(c.Context(), payload.Data.Customer.ID)
	case "subscription.canceled":
		handleErr = h.billingSvc.HandleSubscriptionCanceled(c.Context(), payload.Data.Customer.ID)
	}
	if handleErr != nil {
		// Logged, not propagated: Pagar.me retries non-2xx responses, and
		// an unknown customer id or a repeat delivery isn't something a
		// retry would fix.
		log.Printf("pagarme webhook: %s: %v", payload.Type, handleErr)
	}

	return c.JSON(fiber.Map{"received": true})
}

func (h *BillingHandler) verifyWebhookAuth(c *fiber.Ctx) bool {
	if h.webhookUser == "" && h.webhookPass == "" {
		// No credentials configured for this deployment — Pagar.me's own
		// docs describe webhook auth as optional but recommended.
		return true
	}
	user, pass := parseBasicAuth(c.Get("Authorization"))
	return subtle.ConstantTimeCompare([]byte(user), []byte(h.webhookUser)) == 1 &&
		subtle.ConstantTimeCompare([]byte(pass), []byte(h.webhookPass)) == 1
}

func parseBasicAuth(header string) (user, pass string) {
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return "", ""
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return "", ""
	}
	user, pass, _ = strings.Cut(string(decoded), ":")
	return user, pass
}
