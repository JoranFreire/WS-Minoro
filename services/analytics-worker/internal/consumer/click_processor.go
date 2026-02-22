package consumer

import (
	"context"
	"log"

	"github.com/ws-minoro/analytics-worker/internal/aggregator"
	"github.com/ws-minoro/analytics-worker/internal/parser"
	"github.com/ws-minoro/analytics-worker/internal/writer"
)

type ClickProcessor struct {
	cassandra *writer.CassandraWriter
	pg        *writer.PGWriter
	redis     *aggregator.RedisAggregator
}

func NewClickProcessor(c *writer.CassandraWriter, pg *writer.PGWriter, r *aggregator.RedisAggregator) *ClickProcessor {
	return &ClickProcessor{cassandra: c, pg: pg, redis: r}
}

func (p *ClickProcessor) Process(ctx context.Context, event ClickEvent) error {
	deviceInfo := parser.ParseUserAgent(event.UserAgent)

	click := writer.ClickRecord{
		LinkID:         event.LinkID,
		TenantID:       event.TenantID,
		ShortCode:      event.ShortCode,
		DestinationURL: event.DestinationURL,
		IPHash:         parser.HashIP(event.IP),
		DeviceType:     deviceInfo.DeviceType,
		Browser:        deviceInfo.Browser,
		OS:             deviceInfo.OS,
		Country:        event.Country,
		Referer:        event.Referer,
		Timestamp:      event.Timestamp,
	}

	if err := p.cassandra.WriteClick(ctx, click); err != nil {
		log.Printf("cassandra write error: %v", err)
	}

	if err := p.redis.IncrClick(ctx, event.LinkID, event.TenantID, event.Timestamp); err != nil {
		log.Printf("redis aggregation error: %v", err)
	}

	// Phase 3/4: persist aggregates to PostgreSQL for analytics API.
	if err := p.pg.IncrClickAggregate(ctx, event.LinkID, event.TenantID, event.Timestamp); err != nil {
		log.Printf("pg aggregate error: %v", err)
	}

	if err := p.pg.IncrClickByCountry(ctx, event.LinkID, event.Country, event.Timestamp); err != nil {
		log.Printf("pg country error: %v", err)
	}

	if err := p.pg.IncrClickByDevice(ctx, event.LinkID, deviceInfo.DeviceType, event.Timestamp); err != nil {
		log.Printf("pg device error: %v", err)
	}

	if err := p.pg.IncrQuotaUsage(ctx, event.TenantID, event.Timestamp); err != nil {
		log.Printf("quota update error: %v", err)
	}

	return nil
}
