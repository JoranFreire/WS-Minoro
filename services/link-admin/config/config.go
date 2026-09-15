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
	JWTSecret      string
	FrontendOrigin string
	CookieSecure   bool
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Port:           getEnv("PORT", "8081"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/wsminoro?sslmode=disable"),
		JWTSecret:      requireJWTSecret(),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		CookieSecure:   getEnvBool("COOKIE_SECURE", false),
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
