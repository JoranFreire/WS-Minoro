package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DefaultDomain string // Phase 6: used to detect white-label custom domains
	DatabaseURL   string
	RedisURL      string
	KafkaBrokers  []string
	KafkaTopic    string
	RateLimit     RateLimitConfig
	Invite        InviteConfig
}

type RateLimitConfig struct {
	MaxRequests int
	WindowSecs  int
}

type InviteConfig struct {
	MaxRiskScore            float64
	HealthMonitorIntervalSec int
	HealthTopic             string
	AutoDisabledTopic       string
}

func Load() *Config {
	_ = godotenv.Load()

	maxReqs, _ := strconv.Atoi(getEnv("RATE_LIMIT_MAX_REQUESTS", "100"))
	windowSecs, _ := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW_SECS", "60"))
	maxRisk, _ := strconv.ParseFloat(getEnv("INVITE_MAX_RISK_SCORE", "0.7"), 64)
	healthInterval, _ := strconv.Atoi(getEnv("INVITE_HEALTH_INTERVAL_SEC", "30"))

	return &Config{
		Port:          getEnv("PORT", "8080"),
		DefaultDomain: getEnv("DEFAULT_DOMAIN", "minoro.witsense.com.br"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/wsminoro?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379"),
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "click.raw"),
		RateLimit: RateLimitConfig{
			MaxRequests: maxReqs,
			WindowSecs:  windowSecs,
		},
		Invite: InviteConfig{
			MaxRiskScore:            maxRisk,
			HealthMonitorIntervalSec: healthInterval,
			HealthTopic:             getEnv("KAFKA_HEALTH_TOPIC", "link.health.changed"),
			AutoDisabledTopic:       getEnv("KAFKA_AUTO_DISABLED_TOPIC", "link.auto.disabled"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
