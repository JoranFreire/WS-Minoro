package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/repository"
)

type AnalyticsHandler struct {
	repo *repository.Repository
}

func NewAnalyticsHandler(repo *repository.Repository) *AnalyticsHandler {
	return &AnalyticsHandler{repo: repo}
}

// parseTimeRange reads optional `from` / `to` query params (YYYY-MM-DD).
// Defaults to the last 30 days.
func parseTimeRange(c *fiber.Ctx) (from, to time.Time) {
	now := time.Now().UTC()
	to = now
	from = now.AddDate(0, 0, -30)

	if f := c.Query("from"); f != "" {
		if t, err := time.Parse("2006-01-02", f); err == nil {
			from = t
		}
	}
	if t := c.Query("to"); t != "" {
		if parsed, err := time.Parse("2006-01-02", t); err == nil {
			to = parsed.Add(24 * time.Hour)
		}
	}
	return
}

// GetTimeSeries handles GET /api/v1/analytics/links/:id?granularity=day|hour
func (h *AnalyticsHandler) GetTimeSeries(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	linkID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid link id"})
	}

	granularity := c.Query("granularity", "day")
	if granularity != "hour" && granularity != "day" {
		granularity = "day"
	}

	from, to := parseTimeRange(c)

	if _, err := h.repo.GetLinkByID(c.Context(), linkID, tenantID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "link not found"})
	}

	data, err := h.repo.GetClickTimeSeries(c.Context(), linkID, from, to, granularity)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if data == nil {
		data = []repository.ClickAggregate{}
	}
	return c.JSON(fiber.Map{"data": data, "from": from, "to": to, "granularity": granularity})
}

// GetCountries handles GET /api/v1/analytics/links/:id/countries
func (h *AnalyticsHandler) GetCountries(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	linkID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid link id"})
	}

	from, to := parseTimeRange(c)

	if _, err := h.repo.GetLinkByID(c.Context(), linkID, tenantID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "link not found"})
	}

	data, err := h.repo.GetClicksByCountry(c.Context(), linkID, from, to)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if data == nil {
		data = []repository.ClickByCountry{}
	}
	return c.JSON(fiber.Map{"data": data})
}

// GetDevices handles GET /api/v1/analytics/links/:id/devices
func (h *AnalyticsHandler) GetDevices(c *fiber.Ctx) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	linkID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid link id"})
	}

	from, to := parseTimeRange(c)

	if _, err := h.repo.GetLinkByID(c.Context(), linkID, tenantID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "link not found"})
	}

	data, err := h.repo.GetClicksByDevice(c.Context(), linkID, from, to)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if data == nil {
		data = []repository.ClickByDevice{}
	}
	return c.JSON(fiber.Map{"data": data})
}
