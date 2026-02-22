package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ─── Analytics ───────────────────────────────────────────────

type ClickAggregate struct {
	PeriodStart time.Time `json:"period_start"`
	Clicks      int64     `json:"clicks"`
}

type ClickByCountry struct {
	CountryCode string `json:"country_code"`
	Clicks      int64  `json:"clicks"`
}

type ClickByDevice struct {
	DeviceType string `json:"device_type"`
	Clicks     int64  `json:"clicks"`
}

// GetClickTimeSeries returns hourly or daily click aggregates for a link.
func (r *Repository) GetClickTimeSeries(
	ctx context.Context,
	linkID uuid.UUID,
	from, to time.Time,
	granularity string,
) ([]ClickAggregate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT period_start, total_clicks
		FROM click_aggregates
		WHERE link_id = $1
		  AND granularity = $2
		  AND period_start >= $3
		  AND period_start < $4
		ORDER BY period_start ASC
	`, linkID, granularity, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ClickAggregate
	for rows.Next() {
		var a ClickAggregate
		if err := rows.Scan(&a.PeriodStart, &a.Clicks); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, nil
}

// GetClicksByCountry returns click counts grouped by country for a link.
func (r *Repository) GetClicksByCountry(
	ctx context.Context,
	linkID uuid.UUID,
	from, to time.Time,
) ([]ClickByCountry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT country_code, SUM(total_clicks) AS clicks
		FROM click_by_country
		WHERE link_id = $1
		  AND date >= $2::date
		  AND date < $3::date
		GROUP BY country_code
		ORDER BY clicks DESC
		LIMIT 20
	`, linkID, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ClickByCountry
	for rows.Next() {
		var c ClickByCountry
		if err := rows.Scan(&c.CountryCode, &c.Clicks); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

// GetClicksByDevice returns click counts grouped by device type for a link.
func (r *Repository) GetClicksByDevice(
	ctx context.Context,
	linkID uuid.UUID,
	from, to time.Time,
) ([]ClickByDevice, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT device_type, SUM(total_clicks) AS clicks
		FROM click_by_device
		WHERE link_id = $1
		  AND date >= $2::date
		  AND date < $3::date
		GROUP BY device_type
		ORDER BY clicks DESC
	`, linkID, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ClickByDevice
	for rows.Next() {
		var d ClickByDevice
		if err := rows.Scan(&d.DeviceType, &d.Clicks); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}
