package event

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
	Country        string    `json:"country"`        // Phase 6: geo
	ExperimentID   string    `json:"experiment_id"`  // Phase 6: A/B
}

type KafkaPublisher struct {
	writer *kafka.Writer
	events chan ClickEvent
	done   chan struct{}
}

func NewKafkaPublisher(brokers []string, topic string) *KafkaPublisher {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		Async:        true,
		BatchTimeout: 10 * time.Millisecond,
	}

	p := &KafkaPublisher{
		writer: w,
		events: make(chan ClickEvent, 10000),
		done:   make(chan struct{}),
	}

	go p.processEvents()
	return p
}

func (p *KafkaPublisher) Publish(event ClickEvent) {
	select {
	case p.events <- event:
	default:
		// Drop if buffer full — fire-and-forget
	}
}

func (p *KafkaPublisher) processEvents() {
	defer close(p.done)
	for event := range p.events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := p.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(event.ShortCode),
			Value: data,
		}); err != nil {
			log.Printf("kafka publish error: %v", err)
		}
		cancel()
	}
}

func (p *KafkaPublisher) Close() {
	close(p.events)
	<-p.done
	p.writer.Close()
}
