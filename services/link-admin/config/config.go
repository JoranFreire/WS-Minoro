package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// minJWTSecretLen mirrors the guidance in .env.example — short secrets are
// brute-forceable and must never be allowed to boot the service silently.
const minJWTSecretLen = 32

type Config struct {
	Port           string
	DatabaseURL    string
	RedisURL       string
	JWTSecret      string
	FrontendOrigin string
	CookieSecure   bool

	// Billing (Pagar.me) — all optional. Leaving PagarmeSecretKey unset
	// disables billing entirely (BillingService.Configured() returns
	// false) rather than failing to start, so a deployment that doesn't
	// sell subscriptions yet is unaffected.
	PagarmeSecretKey      string
	PagarmeWebhookUser    string
	PagarmeWebhookPass    string
	PagarmePlanIDStarter  string
	PagarmePlanIDPro      string
	PagarmePlanIDBusiness string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Port:           getEnv("PORT", "8081"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/wsminoro?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:      requireJWTSecret(),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		CookieSecure:   getEnvBool("COOKIE_SECURE", false),

		PagarmeSecretKey:      getEnv("PAGARME_SECRET_KEY", ""),
		PagarmeWebhookUser:    getEnv("PAGARME_WEBHOOK_USER", ""),
		PagarmeWebhookPass:    getEnv("PAGARME_WEBHOOK_PASSWORD", ""),
		PagarmePlanIDStarter:  getEnv("PAGARME_PLAN_ID_STARTER", ""),
		PagarmePlanIDPro:      getEnv("PAGARME_PLAN_ID_PRO", ""),
		PagarmePlanIDBusiness: getEnv("PAGARME_PLAN_ID_BUSINESS", ""),
	}
}

// requireJWTSecret refuses to start with no secret, or one too short to
// resist brute-forcing — there is no insecure default to fall back on.
func requireJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < minJWTSecretLen {
		log.Fatalf("JWT_SECRET must be set and at least %d characters long (see .env.example)", minJWTSecretLen)
	}
	return secret
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}
