package handler

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ws-minoro/link-router/internal/event"
	"github.com/ws-minoro/link-router/internal/ipquality"
	"github.com/ws-minoro/link-router/internal/ratelimit"
	"github.com/ws-minoro/link-router/internal/resolver"
)

type RedirectHandler struct {
	resolver    *resolver.LinkResolver
	publisher   *event.KafkaPublisher
	rateLimiter *ratelimit.RateLimiter
}

func NewRedirectHandler(r *resolver.LinkResolver, p *event.KafkaPublisher, rl *ratelimit.RateLimiter) *RedirectHandler {
	return &RedirectHandler{resolver: r, publisher: p, rateLimiter: rl}
}

func (h *RedirectHandler) Handle(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")
	ip := c.IP()
	ctx := c.Context()

	// Phase 6: bot detection — bots don't consume quota or click slots.
	userAgent := string(c.Request().Header.UserAgent())
	if ipquality.IsBot(userAgent) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}

	allowed, err := h.rateLimiter.Allow(ctx, ip)
	if err != nil {
		log.Printf("rate limiter error: %v", err)
	}
	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "rate limit exceeded"})
	}

	// Phase 6: extract country (Cloudflare or custom proxy header).
	country := c.Get("CF-IPCountry")
	if country == "" {
		country = c.Get("X-Country-Code")
	}

	// Phase 6: extract hostname for white-label custom domain resolution.
	host := c.Hostname()

	destURL, linkID, tenantID, experimentID, err := h.resolver.Resolve(ctx, shortCode, host, country)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "link not found"})
	}

	h.publisher.Publish(event.ClickEvent{
		ShortCode:      shortCode,
		DestinationURL: destURL,
		IP:             ip,
		UserAgent:      userAgent,
		Referer:        c.Get("Referer"),
		Timestamp:      time.Now().UTC(),
		TenantID:       tenantID,
		LinkID:         linkID,
		Country:        country,
		ExperimentID:   experimentID,
	})

	return c.Redirect(destURL, fiber.StatusFound)
}
