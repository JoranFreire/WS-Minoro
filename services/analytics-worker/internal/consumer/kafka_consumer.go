package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type ClickEvent struct {
	ShortCode      string    `json:"short_code"`
	DestinationURL string    `json:"destination_url"`
	IP             string    `json:"ip"`
	UserAgent      string    `json:"user_agent"`
	Referer        string    `json:"referer"`
	Timestamp      time.Time `json:"timestamp"`
	TenantID       string    `json:"tenant_id"`
	LinkID         string    `json:"link_id"`
	Country        string    `json:"country"`       // Phase 6
	ExperimentID   string    `json:"experiment_id"` // Phase 6
}

type Processor interface {
	Process(ctx context.Context, event ClickEvent) error
}

type KafkaConsumer struct {
	reader    *kafka.Reader
	processor Processor
}

func NewKafkaConsumer(brokers []string, topic, groupID string, processor Processor) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
	return &KafkaConsumer{reader: reader, processor: processor}
}

func (c *KafkaConsumer) Run(ctx context.Context) error {
	defer c.reader.Close()
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("kafka read error: %v", err)
			continue
		}

		var event ClickEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("unmarshal error: %v", err)
			continue
		}

		if err := c.processor.Process(ctx, event); err != nil {
			log.Printf("process error: %v", err)
		}
	}
}
