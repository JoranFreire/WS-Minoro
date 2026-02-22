package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/middleware"
	"github.com/ws-minoro/link-admin/internal/service"
)

type LinkHandler struct {
	linkSvc *service.LinkService
}

func NewLinkHandler(linkSvc *service.LinkService) *LinkHandler {
	return &LinkHandler{linkSvc: linkSvc}
}

func getTenantID(c *fiber.Ctx) (uuid.UUID, error) {
	claims, ok := c.Locals(middleware.UserContextKey).(*service.Claims)
	if !ok {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	return uuid.Parse(claims.TenantID)
}

func (h *LinkHandler) List(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	links, err := h.linkSvc.List(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if links == nil {
		return c.JSON([]any{})
	}
	return c.JSON(links)
}

func (h *LinkHandler) Get(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	link, dests, err := h.linkSvc.Get(c.Context(), id, tenantID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(fiber.Map{"link": link, "destinations": dests})
}

func (h *LinkHandler) Create(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var req struct {
		Title           string `json:"title"`
		FallbackURL     string `json:"fallback_url"`
		RoutingStrategy string `json:"routing_strategy"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.RoutingStrategy == "" {
		req.RoutingStrategy = "round_robin"
	}
	link, err := h.linkSvc.Create(c.Context(), tenantID, req.Title, req.FallbackURL, req.RoutingStrategy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(link)
}

func (h *LinkHandler) Update(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var req struct {
		Title           string `json:"title"`
		FallbackURL     string `json:"fallback_url"`
		RoutingStrategy string `json:"routing_strategy"`
		IsActive        bool   `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	link, err := h.linkSvc.Update(c.Context(), id, tenantID, req.Title, req.FallbackURL, req.RoutingStrategy, req.IsActive)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(link)
}

func (h *LinkHandler) Delete(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.linkSvc.Delete(c.Context(), id, tenantID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *LinkHandler) AddDestination(c *fiber.Ctx) error {
	linkID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid link id"})
	}
	var req struct {
		URL       string `json:"url"`
		Weight    int    `json:"weight"`
		MaxClicks *int   `json:"max_clicks"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Weight == 0 {
		req.Weight = 1
	}
	dest, err := h.linkSvc.AddDestination(c.Context(), linkID, req.URL, req.Weight, req.MaxClicks)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(dest)
}

func (h *LinkHandler) UpdateDestination(c *fiber.Ctx) error {
	linkID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid link id"})
	}
	destID, err := uuid.Parse(c.Params("destId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid destination id"})
	}
	var req struct {
		URL       string `json:"url"`
		Weight    int    `json:"weight"`
		MaxClicks *int   `json:"max_clicks"`
		IsActive  bool   `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := h.linkSvc.UpdateDestination(c.Context(), destID, linkID, req.URL, req.Weight, req.MaxClicks, req.IsActive); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *LinkHandler) DeleteDestination(c *fiber.Ctx) error {
	linkID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid link id"})
	}
	destID, err := uuid.Parse(c.Params("destId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid destination id"})
	}
	if err := h.linkSvc.DeleteDestination(c.Context(), destID, linkID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
