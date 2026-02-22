package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/middleware"
	"github.com/ws-minoro/link-admin/internal/service"
)

type APIKeyHandler struct {
	tenantSvc *service.TenantService
}

func NewAPIKeyHandler(tenantSvc *service.TenantService) *APIKeyHandler {
	return &APIKeyHandler{tenantSvc: tenantSvc}
}

func (h *APIKeyHandler) Create(c *fiber.Ctx) error {
	claims, ok := c.Locals(middleware.UserContextKey).(*service.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	tenantID, _ := uuid.Parse(claims.TenantID)
	var req struct {
		Label string `json:"label"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	rawKey, err := h.tenantSvc.CreateAPIKey(c.Context(), tenantID, req.Label)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"key":     rawKey,
		"message": "store this key safely, it will not be shown again",
	})
}

func (h *APIKeyHandler) Delete(c *fiber.Ctx) error {
	claims, ok := c.Locals(middleware.UserContextKey).(*service.Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	tenantID, _ := uuid.Parse(claims.TenantID)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.tenantSvc.DeleteAPIKey(c.Context(), id, tenantID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
