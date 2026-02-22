package writer

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PGWriter struct {
	pool *pgxpool.Pool
}

func NewPGWriter(databaseURL string) *PGWriter {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		panic("failed to connect to postgres: " + err.Error())
	}
	return &PGWriter{pool: pool}
}

// IncrClickAggregate upserts the hourly and daily click aggregate for a link.
func (w *PGWriter) IncrClickAggregate(ctx context.Context, linkID, tenantID string, ts time.Time) error {
	hourStart := ts.Truncate(time.Hour)
	dayStart := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)

	for _, row := range []struct {
		period      time.Time
		granularity string
	}{
		{hourStart, "hour"},
		{dayStart, "day"},
	} {
		_, err := w.pool.Exec(ctx, `
			INSERT INTO click_aggregates (link_id, tenant_id, period_start, granularity, total_clicks)
			VALUES ($1, $2, $3, $4, 1)
			ON CONFLICT (link_id, period_start, granularity)
			DO UPDATE SET total_clicks = click_aggregates.total_clicks + 1
		`, linkID, tenantID, row.period, row.granularity)
		if err != nil {
			return err
		}
	}
	return nil
}

// IncrClickByCountry upserts the daily country click count for a link.
func (w *PGWriter) IncrClickByCountry(ctx context.Context, linkID, countryCode string, ts time.Time) error {
	if countryCode == "" {
		countryCode = "XX"
	}
	date := ts.Format("2006-01-02")
	_, err := w.pool.Exec(ctx, `
		INSERT INTO click_by_country (link_id, date, country_code, total_clicks)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (link_id, date, country_code)
		DO UPDATE SET total_clicks = click_by_country.total_clicks + 1
	`, linkID, date, countryCode)
	return err
}

// IncrClickByDevice upserts the daily device click count for a link.
func (w *PGWriter) IncrClickByDevice(ctx context.Context, linkID, deviceType string, ts time.Time) error {
	if deviceType == "" {
		deviceType = "unknown"
	}
	date := ts.Format("2006-01-02")
	_, err := w.pool.Exec(ctx, `
		INSERT INTO click_by_device (link_id, date, device_type, total_clicks)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (link_id, date, device_type)
		DO UPDATE SET total_clicks = click_by_device.total_clicks + 1
	`, linkID, date, deviceType)
	return err
}

// IncrQuotaUsage upserts the monthly quota counter for a tenant.
func (w *PGWriter) IncrQuotaUsage(ctx context.Context, tenantID string, month time.Time) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO quota_usage (tenant_id, month, clicks_used)
		VALUES ($1, date_trunc('month', $2::date), 1)
		ON CONFLICT (tenant_id, month)
		DO UPDATE SET clicks_used = quota_usage.clicks_used + 1
	`, tenantID, month.Format("2006-01-02"))
	return err
}
