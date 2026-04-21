package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/ws-minoro/link-admin/config"
	"github.com/ws-minoro/link-admin/internal/handler"
	"github.com/ws-minoro/link-admin/internal/middleware"
	"github.com/ws-minoro/link-admin/internal/repository"
	"github.com/ws-minoro/link-admin/internal/service"
)

func main() {
	cfg := config.Load()

	repo := repository.NewRepository(cfg.DatabaseURL)

	linkSvc := service.NewLinkService(repo)
	tenantSvc := service.NewTenantService(repo)
	authSvc := service.NewAuthService(repo, cfg.JWTSecret)

	linkHandler := handler.NewLinkHandler(linkSvc)
	tenantHandler := handler.NewTenantHandler(tenantSvc)
	authHandler := handler.NewAuthHandler(authSvc)
	apikeyHandler := handler.NewAPIKeyHandler(tenantSvc)
	analyticsHandler := handler.NewAnalyticsHandler(repo)

	authMw := middleware.NewAuthMiddleware(authSvc)

	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	auth := app.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)

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

	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
