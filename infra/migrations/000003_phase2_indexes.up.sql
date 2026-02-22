-- Indexes to support Phase 2 health monitor queries and invite routing.

-- Speed up GetExpiredCooldownDestinations (cooldown monitor scans this frequently).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_link_destinations_expired_cooldown
    ON link_destinations (cooldown_until)
    WHERE is_active = false AND cooldown_until IS NOT NULL;

-- Speed up filtering active destinations ordered by weight for weighted routing.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_link_destinations_active_link
    ON link_destinations (link_id, is_active)
    INCLUDE (weight, max_clicks, current_clicks, cooldown_until, risk_score);

-- Speed up risk score filtering per link.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_link_destinations_risk_score
    ON link_destinations (link_id, risk_score)
    WHERE is_active = true;
