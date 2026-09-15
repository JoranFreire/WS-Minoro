package consumer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ws-minoro/analytics-worker/internal/health"
	"github.com/ws-minoro/analytics-worker/internal/writer"
)

type fakeCassandraWriter struct {
	mu       sync.Mutex
	writes   []writer.ClickRecord
	writeErr error
}

func (f *fakeCassandraWriter) WriteClick(ctx context.Context, click writer.ClickRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeErr != nil {
		return f.writeErr
	}
	f.writes = append(f.writes, click)
	return nil
}

type fakePGWriter struct {
	mu             sync.Mutex
	aggregateCalls int
	countryCalls   int
	deviceCalls    int
	quotaCalls     int
	aggregateErr   error
	countryErr     error
	deviceErr      error
	quotaErr       error
}

func (f *fakePGWriter) IncrClickAggregate(ctx context.Context, linkID, tenantID string, ts time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.aggregateCalls++
	return f.aggregateErr
}

func (f *fakePGWriter) IncrClickByCountry(ctx context.Context, linkID, countryCode string, ts time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.countryCalls++
	return f.countryErr
}

func (f *fakePGWriter) IncrClickByDevice(ctx context.Context, linkID, deviceType string, ts time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deviceCalls++
	return f.deviceErr
}

func (f *fakePGWriter) IncrQuotaUsage(ctx context.Context, tenantID string, month time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.quotaCalls++
	return f.quotaErr
}

type fakeRedisAggregator struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeRedisAggregator) IncrClick(ctx context.Context, linkID, tenantID string, ts time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.err
}

func newTestEvent() ClickEvent {
	return ClickEvent{
		ShortCode:      "abc123",
		DestinationURL: "https://example.com",
		IP:             "203.0.113.42",
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		Referer:        "https://google.com",
		Timestamp:      time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC),
		TenantID:       "tenant-1",
		LinkID:         "link-1",
		Country:        "BR",
	}
}

func TestClickProcessor_Process_AllSucceed(t *testing.T) {
	cass := &fakeCassandraWriter{}
	pg := &fakePGWriter{}
	redis := &fakeRedisAggregator{}
	tracker := health.NewTracker()
	p := NewClickProcessor(cass, pg, redis, tracker)

	if err := p.Process(context.Background(), newTestEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cass.writes) != 1 {
		t.Fatalf("expected 1 cassandra write, got %d", len(cass.writes))
	}
	if redis.calls != 1 {
		t.Fatalf("expected 1 redis call, got %d", redis.calls)
	}
	if pg.aggregateCalls != 1 || pg.countryCalls != 1 || pg.deviceCalls != 1 || pg.quotaCalls != 1 {
		t.Fatalf("expected every pg write called once, got %+v", pg)
	}
	if !tracker.Healthy() {
		t.Fatal("expected tracker to remain healthy after a fully successful process")
	}
}

func TestClickProcessor_Process_PartialFailureReturnsErrorButStaysHealthy(t *testing.T) {
	cass := &fakeCassandraWriter{writeErr: errors.New("cassandra down")}
	pg := &fakePGWriter{}
	redis := &fakeRedisAggregator{}
	tracker := health.NewTracker()
	p := NewClickProcessor(cass, pg, redis, tracker)

	err := p.Process(context.Background(), newTestEvent())
	if err == nil {
		t.Fatal("expected an error when a single sink fails")
	}

	// Other sinks should still have been attempted — one failure must not
	// short-circuit the rest.
	if redis.calls != 1 || pg.aggregateCalls != 1 {
		t.Fatalf("expected other sinks to still be attempted, redis=%d pg.aggregate=%d", redis.calls, pg.aggregateCalls)
	}

	if !tracker.Healthy() {
		t.Fatal("a single failing sink should not flip the tracker unhealthy")
	}
}

func TestClickProcessor_Process_TotalFailureRecordsTrackerFailure(t *testing.T) {
	failErr := errors.New("down")
	cass := &fakeCassandraWriter{writeErr: failErr}
	pg := &fakePGWriter{aggregateErr: failErr, countryErr: failErr, deviceErr: failErr, quotaErr: failErr}
	redis := &fakeRedisAggregator{err: failErr}
	tracker := health.NewTracker()
	p := NewClickProcessor(cass, pg, redis, tracker)

	err := p.Process(context.Background(), newTestEvent())
	if err == nil {
		t.Fatal("expected an error when every sink fails")
	}

	// One event alone shouldn't flip it unhealthy — only sustained failure.
	if !tracker.Healthy() {
		t.Fatal("a single fully-failed event should not yet flip the tracker unhealthy")
	}

	for range 9 {
		_ = p.Process(context.Background(), newTestEvent())
	}

	if tracker.Healthy() {
		t.Fatal("expected tracker to be unhealthy after sustained total failures")
	}
}

func TestClickProcessor_Process_SuccessAfterFailuresRestoresHealth(t *testing.T) {
	failErr := errors.New("down")
	cass := &fakeCassandraWriter{writeErr: failErr}
	pg := &fakePGWriter{aggregateErr: failErr, countryErr: failErr, deviceErr: failErr, quotaErr: failErr}
	redis := &fakeRedisAggregator{err: failErr}
	tracker := health.NewTracker()
	p := NewClickProcessor(cass, pg, redis, tracker)

	for range 9 {
		_ = p.Process(context.Background(), newTestEvent())
	}

	cass.writeErr = nil
	pg.aggregateErr, pg.countryErr, pg.deviceErr, pg.quotaErr = nil, nil, nil, nil
	redis.err = nil

	if err := p.Process(context.Background(), newTestEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tracker.Healthy() {
		t.Fatal("expected a fully successful process to restore health")
	}
}

func TestClickProcessor_Process_HashesIPBeforeWritingToCassandra(t *testing.T) {
	cass := &fakeCassandraWriter{}
	pg := &fakePGWriter{}
	redis := &fakeRedisAggregator{}
	tracker := health.NewTracker()
	p := NewClickProcessor(cass, pg, redis, tracker)

	event := newTestEvent()
	if err := p.Process(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cass.writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(cass.writes))
	}
	if cass.writes[0].IPHash == event.IP {
		t.Fatal("expected the raw IP to never reach the Cassandra write")
	}
	if cass.writes[0].IPHash == "" {
		t.Fatal("expected a non-empty IP hash")
	}
}
