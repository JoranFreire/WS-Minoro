package consumer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ws-minoro/analytics-worker/internal/health"
	"github.com/ws-minoro/analytics-worker/internal/parser"
	"github.com/ws-minoro/analytics-worker/internal/writer"
)

// writeCount is how many independent sink writes Process attempts per
// event — used to tell "every sink failed" (systemic outage) apart from a
// single flaky write.
const writeCount = 6

// CassandraWriter, PGWriter and RedisAggregator are the subsets of their
// concrete counterparts the processor depends on. Defined here (rather than
// depended on directly) so tests can inject fakes instead of requiring a
// live Cassandra/Postgres/Redis.
type CassandraWriter interface {
	WriteClick(ctx context.Context, click writer.ClickRecord) error
}

type PGWriter interface {
	IncrClickAggregate(ctx context.Context, linkID, tenantID string, ts time.Time) error
	IncrClickByCountry(ctx context.Context, linkID, countryCode string, ts time.Time) error
	IncrClickByDevice(ctx context.Context, linkID, deviceType string, ts time.Time) error
	IncrQuotaUsage(ctx context.Context, tenantID string, month time.Time) error
}

type RedisAggregator interface {
	IncrClick(ctx context.Context, linkID, tenantID string, ts time.Time) error
}

type ClickProcessor struct {
	cassandra CassandraWriter
	pg        PGWriter
	redis     RedisAggregator
	tracker   *health.Tracker
}

func NewClickProcessor(c CassandraWriter, pg PGWriter, r RedisAggregator, tracker *health.Tracker) *ClickProcessor {
	return &ClickProcessor{cassandra: c, pg: pg, redis: r, tracker: tracker}
}

// Process persists a click event to every sink and reports every failure —
// unlike a version that logs and swallows errors, so a full outage across
// every downstream store is neither silently dropped nor left unreported.
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

	var errs []error

	if err := p.cassandra.WriteClick(ctx, click); err != nil {
		errs = append(errs, fmt.Errorf("cassandra write: %w", err))
	}

	if err := p.redis.IncrClick(ctx, event.LinkID, event.TenantID, event.Timestamp); err != nil {
		errs = append(errs, fmt.Errorf("redis aggregation: %w", err))
	}

	// Phase 3/4: persist aggregates to PostgreSQL for analytics API.
	if err := p.pg.IncrClickAggregate(ctx, event.LinkID, event.TenantID, event.Timestamp); err != nil {
		errs = append(errs, fmt.Errorf("pg aggregate: %w", err))
	}

	if err := p.pg.IncrClickByCountry(ctx, event.LinkID, event.Country, event.Timestamp); err != nil {
		errs = append(errs, fmt.Errorf("pg country: %w", err))
	}

	if err := p.pg.IncrClickByDevice(ctx, event.LinkID, deviceInfo.DeviceType, event.Timestamp); err != nil {
		errs = append(errs, fmt.Errorf("pg device: %w", err))
	}

	if err := p.pg.IncrQuotaUsage(ctx, event.TenantID, event.Timestamp); err != nil {
		errs = append(errs, fmt.Errorf("quota update: %w", err))
	}

	if len(errs) == writeCount {
		p.tracker.RecordFailure()
	} else {
		p.tracker.RecordSuccess()
	}

	return errors.Join(errs...)
}
