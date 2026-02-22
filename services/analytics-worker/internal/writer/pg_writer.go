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

func (w *PGWriter) IncrClickAggregate(ctx context.Context, linkID, tenantID string, periodStart time.Time, granularity string) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO click_aggregates (link_id, tenant_id, period_start, granularity, total_clicks)
		VALUES ($1, $2, $3, $4, 1)
		ON CONFLICT (link_id, period_start, granularity)
		DO UPDATE SET total_clicks = click_aggregates.total_clicks + 1
	`, linkID, tenantID, periodStart, granularity)
	return err
}

func (w *PGWriter) IncrQuotaUsage(ctx context.Context, tenantID string, month time.Time) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO quota_usage (tenant_id, month, clicks_used)
		VALUES ($1, date_trunc('month', $2::date), 1)
		ON CONFLICT (tenant_id, month)
		DO UPDATE SET clicks_used = quota_usage.clicks_used + 1
	`, tenantID, month.Format("2006-01-02"))
	return err
}
