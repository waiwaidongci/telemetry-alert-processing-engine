CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    serial_number TEXT NOT NULL,
    location TEXT NOT NULL DEFAULT '',
    tags JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL CHECK (status IN ('active', 'inactive')),
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (tenant_id, serial_number)
);

CREATE INDEX IF NOT EXISTS idx_devices_tenant_status ON devices(tenant_id, status);

CREATE TABLE IF NOT EXISTS telemetry_points (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    received_at TIMESTAMPTZ NOT NULL,
    idempotency_key TEXT,
    CONSTRAINT uq_telemetry_idempotency UNIQUE (tenant_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_telemetry_query ON telemetry_points(tenant_id, device_id, metric_name, timestamp);

CREATE TABLE IF NOT EXISTS telemetry_aggregates (
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    granularity TEXT NOT NULL CHECK (granularity IN ('1m', '5m', '1h', '1d')),
    bucket_time TIMESTAMPTZ NOT NULL,
    count BIGINT NOT NULL,
    sum DOUBLE PRECISION NOT NULL,
    min DOUBLE PRECISION NOT NULL,
    max DOUBLE PRECISION NOT NULL,
    PRIMARY KEY (tenant_id, device_id, metric_name, granularity, bucket_time)
);

CREATE TABLE IF NOT EXISTS alert_rules (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    device_id TEXT REFERENCES devices(id) ON DELETE CASCADE,
    device_tag TEXT NOT NULL DEFAULT '',
    metric_name TEXT NOT NULL,
    operator TEXT NOT NULL CHECK (operator IN ('gt', 'gte', 'lt', 'lte', 'eq')),
    threshold DOUBLE PRECISION NOT NULL,
    window_ms BIGINT NOT NULL,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    level TEXT NOT NULL CHECK (level IN ('info', 'warning', 'critical')),
    cooldown_ms BIGINT NOT NULL DEFAULT 0,
    channels JSONB NOT NULL DEFAULT '[]'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_tenant_enabled ON alert_rules(tenant_id, enabled);

CREATE TABLE IF NOT EXISTS alert_states (
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rule_id TEXT NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('normal', 'pending', 'triggered', 'recovered')),
    since TIMESTAMPTZ,
    last_value DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_event_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, rule_id, device_id, metric_name)
);

CREATE TABLE IF NOT EXISTS alert_events (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rule_id TEXT NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    rule_name TEXT NOT NULL,
    device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('triggered', 'continuing', 'recovered')),
    level TEXT NOT NULL CHECK (level IN ('info', 'warning', 'critical')),
    channels JSONB NOT NULL DEFAULT '[]'::jsonb,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alert_events_query ON alert_events(tenant_id, created_at);

CREATE TABLE IF NOT EXISTS notification_attempts (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL REFERENCES alert_events(id) ON DELETE CASCADE,
    channel TEXT NOT NULL CHECK (channel IN ('webhook', 'log')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notification_pending ON notification_attempts(status, next_attempt_at);
