DROP INDEX CONCURRENTLY IF EXISTS idx_link_destinations_country;
DROP INDEX CONCURRENTLY IF EXISTS idx_tenants_custom_domain;
ALTER TABLE link_destinations DROP COLUMN IF EXISTS experiment_id;
ALTER TABLE link_destinations DROP COLUMN IF EXISTS allowed_countries;
