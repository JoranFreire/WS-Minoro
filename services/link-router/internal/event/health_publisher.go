package event

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// HealthEvent is published when a destination's status changes.
type HealthEvent struct {
	DestinationID string    `json:"destination_id"`
	LinkID        string    `json:"link_id"`
	TenantID      string    `json:"tenant_id"`
	ShortCode     string    `json:"short_code"`
	Status        string    `json:"status"` // "disabled" | "reactivated"
	Reason        string    `json:"reason"`
	Timestamp     time.Time `json:"timestamp"`
}

// HealthPublisher publishes health and auto-disabled events to Kafka.
type HealthPublisher struct {
	healthWriter   *kafka.Writer
	disabledWriter *kafka.Writer
}

func NewHealthPublisher(brokers []string, healthTopic, disabledTopic string) *HealthPublisher {
	return &HealthPublisher{
		healthWriter: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        healthTopic,
			Balancer:     &kafka.LeastBytes{},
			Async:        true,
			MaxAttempts:  3,
			BatchTimeout: 10 * time.Millisecond,
		},
		disabledWriter: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        disabledTopic,
			Balancer:     &kafka.LeastBytes{},
			Async:        true,
			MaxAttempts:  3,
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

// PublishDisabled publishes to both link.health.changed and link.auto.disabled topics.
func (p *HealthPublisher) PublishDisabled(ctx context.Context, evt HealthEvent) {
	evt.Status = "disabled"
	p.publish(ctx, p.healthWriter, evt)
	p.publish(ctx, p.disabledWriter, evt)
}

// PublishReactivated publishes a reactivation event to link.health.changed.
func (p *HealthPublisher) PublishReactivated(ctx context.Context, evt HealthEvent) {
	evt.Status = "reactivated"
	p.publish(ctx, p.healthWriter, evt)
}

func (p *HealthPublisher) publish(ctx context.Context, w *kafka.Writer, evt HealthEvent) {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("health_publisher: marshal error: %v", err)
		return
	}
	if err := w.WriteMessages(ctx, kafka.Message{Value: data}); err != nil {
		log.Printf("health_publisher: write error: %v", err)
	}
}

func (p *HealthPublisher) Close() {
	_ = p.healthWriter.Close()
	_ = p.disabledWriter.Close()
}
