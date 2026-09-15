package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/ws-minoro/link-admin/config"
	"github.com/ws-minoro/link-admin/internal/billing"
	"github.com/ws-minoro/link-admin/internal/handler"
	"github.com/ws-minoro/link-admin/internal/middleware"
	"github.com/ws-minoro/link-admin/internal/quota"
	"github.com/ws-minoro/link-admin/internal/repository"
	"github.com/ws-minoro/link-admin/internal/service"
)

func main() {
	cfg := config.Load()

	pool := repository.Connect(cfg.DatabaseURL)

	userRepo := repository.NewUserRepository(pool)
	tenantRepo := repository.NewTenantRepository(pool)
	linkRepo := repository.NewLinkRepository(pool)
	apiKeyRepo := repository.NewAPIKeyRepository(pool)
	analyticsRepo := repository.NewAnalyticsRepository(pool)
	registrationRepo := repository.NewRegistrationRepository(pool)

	linkSvc := service.NewLinkService(linkRepo)
	tenantSvc := service.NewTenantService(tenantRepo, apiKeyRepo)
	authSvc := service.NewAuthService(userRepo, apiKeyRepo, registrationRepo, cfg.JWTSecret)

	// Billing is optional: a nil client leaves BillingService.Configured()
	// false, so a deployment without PAGARME_SECRET_KEY set is unaffected.
	var pagarmeClient service.PagarmeClient
	if cfg.PagarmeSecretKey != "" {
		pagarmeClient = billing.NewClient(cfg.PagarmeSecretKey)
	}
	quotaUnblocker := quota.NewUnblocker(cfg.RedisURL)
	billingSvc := service.NewBillingService(tenantRepo, userRepo, pagarmeClient, quotaUnblocker, map[string]string{
		"starter":  cfg.PagarmePlanIDStarter,
		"pro":      cfg.PagarmePlanIDPro,
		"business": cfg.PagarmePlanIDBusiness,
	})

	linkHandler := handler.NewLinkHandler(linkSvc)
	tenantHandler := handler.NewTenantHandler(tenantSvc)
	authHandler := handler.NewAuthHandler(authSvc, cfg.CookieSecure)
	apikeyHandler := handler.NewAPIKeyHandler(tenantSvc)
	analyticsHandler := handler.NewAnalyticsHandler(linkRepo, analyticsRepo)
	billingHandler := handler.NewBillingHandler(billingSvc, cfg.PagarmeWebhookUser, cfg.PagarmeWebhookPass)

	authMw := middleware.NewAuthMiddleware(authSvc)

	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.FrontendOrigin,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Not under authMw.Authenticate — Pagar.me is the caller, authenticated
	// via its own Basic Auth credentials (see BillingHandler.Webhook), not
	// a user's JWT.
	app.Post("/webhooks/pagarme", billingHandler.Webhook)

	auth := app.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/logout", authHandler.Logout)

	api := app.Group("/api/v1", authMw.Authenticate)

	links := api.Group("/links")
	links.Get("/", linkHandler.List)
	links.Post("/", linkHandler.Create)
	links.Get("/:id", linkHandler.Get)
	links.Put("/:id", linkHandler.Update)
	links.Delete("/:id", linkHandler.Delete)
	links.Post("/:id/destinations", linkHandler.AddDestination)
	links.Put("/:id/destinations/:destId", linkHandler.UpdateDestination)
	links.Delete("/:id/destinations/:destId", linkHandler.DeleteDestination)

	tenants := api.Group("/tenants")
	tenants.Get("/me", tenantHandler.GetMe)
	tenants.Get("/me/quota", tenantHandler.GetQuota)

	apikeys := api.Group("/api-keys")
	apikeys.Post("/", apikeyHandler.Create)
	apikeys.Delete("/:id", apikeyHandler.Delete)

	analytics := api.Group("/analytics/links")
	analytics.Get("/:id", analyticsHandler.GetTimeSeries)
	analytics.Get("/:id/countries", analyticsHandler.GetCountries)
	analytics.Get("/:id/devices", analyticsHandler.GetDevices)

	billingRoutes := api.Group("/billing")
	billingRoutes.Post("/subscribe", billingHandler.Subscribe)
	billingRoutes.Delete("/subscription", billingHandler.Cancel)

	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
