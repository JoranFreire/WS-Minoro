package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/middleware"
	"github.com/ws-minoro/link-admin/internal/service"
)

type TenantHandler struct {
	tenantSvc *service.TenantService
}

func NewTenantHandler(tenantSvc *service.TenantService) *TenantHandler {
	return &TenantHandler{tenantSvc: tenantSvc}
}

func (h *TenantHandler) GetMe(c *fiber.Ctx) error {
	claims, ok := c.Locals(middleware.UserContextKey).(*service.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tenant"})
	}
	tenant, err := h.tenantSvc.GetTenant(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "tenant not found"})
	}
	return c.JSON(tenant)
}

func (h *TenantHandler) GetQuota(c *fiber.Ctx) error {
	claims, ok := c.Locals(middleware.UserContextKey).(*service.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tenant"})
	}
	used, limit, err := h.tenantSvc.GetQuota(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"clicks_used":  used,
		"clicks_limit": limit,
		"remaining":    limit - used,
	})
}
