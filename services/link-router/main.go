package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/ws-minoro/link-router/config"
	"github.com/ws-minoro/link-router/internal/cache"
	"github.com/ws-minoro/link-router/internal/event"
	"github.com/ws-minoro/link-router/internal/handler"
	"github.com/ws-minoro/link-router/internal/health"
	"github.com/ws-minoro/link-router/internal/ratelimit"
	"github.com/ws-minoro/link-router/internal/resolver"
	"github.com/ws-minoro/link-router/internal/store"
)

func main() {
	cfg := config.Load()

	redisCache := cache.NewRedisCache(cfg.RedisURL)
	pgStore := store.NewPGStore(cfg.DatabaseURL)
	kafkaPublisher := event.NewKafkaPublisher(cfg.KafkaBrokers, cfg.KafkaTopic)
	healthPublisher := event.NewHealthPublisher(
		cfg.KafkaBrokers,
		cfg.Invite.HealthTopic,
		cfg.Invite.AutoDisabledTopic,
	)
	rateLimiter := ratelimit.NewRateLimiter(redisCache, ratelimit.Config{
		MaxRequests: cfg.RateLimit.MaxRequests,
		WindowSecs:  cfg.RateLimit.WindowSecs,
	})
	linkResolver := resolver.NewLinkResolver(
		redisCache,
		pgStore,
		healthPublisher,
		cfg.Invite.MaxRiskScore,
	)

	// Phase 2: background health monitor reactivates expired cooldown destinations.
	monitor := health.NewInviteHealthMonitor(
		pgStore,
		redisCache,
		healthPublisher,
		cfg.Invite.HealthMonitorIntervalSec,
	)

	redirectHandler := handler.NewRedirectHandler(linkResolver, kafkaPublisher, rateLimiter)

	app := fiber.New(fiber.Config{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} ${latency} ${method} ${path}\n",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/metrics", handler.MetricsHandler())
	app.Get("/:shortCode", redirectHandler.Handle)

	// Start HTTP server.
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Start health monitor in a cancellable context.
	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	go monitor.Start(monitorCtx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	monitorCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	kafkaPublisher.Close()
	healthPublisher.Close()
	pgStore.Close()
}
