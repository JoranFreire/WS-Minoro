CREATE KEYSPACE IF NOT EXISTS link_analytics
  WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE link_analytics;

CREATE TABLE IF NOT EXISTS clicks_by_link (
  link_id UUID,
  day DATE,
  click_id TIMEUUID,
  destination_url TEXT,
  country TEXT,
  city TEXT,
  device_type TEXT,
  browser TEXT,
  os TEXT,
  referer TEXT,
  ip_hash TEXT,
  tenant_id UUID,
  PRIMARY KEY ((link_id, day), click_id)
) WITH CLUSTERING ORDER BY (click_id DESC)
  AND default_time_to_live = 7776000;

CREATE TABLE IF NOT EXISTS clicks_by_tenant (
  tenant_id UUID,
  day DATE,
  click_id TIMEUUID,
  link_id UUID,
  short_code TEXT,
  country TEXT,
  device_type TEXT,
  PRIMARY KEY ((tenant_id, day), click_id)
) WITH CLUSTERING ORDER BY (click_id DESC)
  AND default_time_to_live = 7776000;
