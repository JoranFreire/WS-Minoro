-- Phase 6: Differentiators

-- Geo routing: whitelist of country codes per destination (empty = all allowed).
ALTER TABLE link_destinations
    ADD COLUMN IF NOT EXISTS allowed_countries TEXT[] NOT NULL DEFAULT '{}';

-- A/B testing: tag destinations with an experiment group name.
ALTER TABLE link_destinations
    ADD COLUMN IF NOT EXISTS experiment_id VARCHAR(50);

-- Index for custom domain lookup (white-label).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tenants_custom_domain
    ON tenants (custom_domain)
    WHERE custom_domain IS NOT NULL AND custom_domain <> '';

-- Index to speed up geo-filtered routing.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_link_destinations_country
    ON link_destinations USING gin (allowed_countries)
    WHERE is_active = true;
