package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ws-minoro/analytics-worker/config"
	"github.com/ws-minoro/analytics-worker/internal/aggregator"
	"github.com/ws-minoro/analytics-worker/internal/consumer"
	"github.com/ws-minoro/analytics-worker/internal/writer"
)

func main() {
	cfg := config.Load()

	cassandraWriter := writer.NewCassandraWriter(cfg.CassandraHosts, cfg.CassandraKeyspace)
	pgWriter := writer.NewPGWriter(cfg.DatabaseURL)
	redisAgg := aggregator.NewRedisAggregator(cfg.RedisURL)

	processor := consumer.NewClickProcessor(cassandraWriter, pgWriter, redisAgg)
	kafkaConsumer := consumer.NewKafkaConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, processor)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cancel()
	}()

	log.Println("analytics-worker started, consuming from", cfg.KafkaTopic)
	if err := kafkaConsumer.Run(ctx); err != nil {
		log.Printf("consumer stopped: %v", err)
	}
}
