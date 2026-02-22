-- Link click aggregates (hourly/daily)
CREATE TABLE click_aggregates (
  link_id UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  period_start TIMESTAMPTZ NOT NULL,
  granularity VARCHAR(10) NOT NULL CHECK (granularity IN ('hour','day')),
  total_clicks BIGINT NOT NULL DEFAULT 0,
  unique_ips BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (link_id, period_start, granularity)
);

CREATE INDEX idx_click_aggregates_tenant ON click_aggregates(tenant_id, period_start);

-- Link click by country
CREATE TABLE click_by_country (
  link_id UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  country_code VARCHAR(2) NOT NULL,
  total_clicks BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (link_id, date, country_code)
);

-- Link click by device
CREATE TABLE click_by_device (
  link_id UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  device_type VARCHAR(50) NOT NULL,
  total_clicks BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (link_id, date, device_type)
);
