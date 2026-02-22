package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL       string
	RedisURL          string
	KafkaBrokers      []string
	KafkaTopic        string
	KafkaGroupID      string
	CassandraHosts    []string
	CassandraKeyspace string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/wsminoro?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379"),
		KafkaBrokers:      strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:        getEnv("KAFKA_TOPIC", "click.raw"),
		KafkaGroupID:      getEnv("KAFKA_GROUP_ID", "analytics-worker"),
		CassandraHosts:    strings.Split(getEnv("CASSANDRA_HOSTS", "localhost"), ","),
		CassandraKeyspace: getEnv("CASSANDRA_KEYSPACE", "link_analytics"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
