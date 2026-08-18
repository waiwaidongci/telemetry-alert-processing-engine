package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

var schema = []string{
	`CREATE TABLE IF NOT EXISTS tenants (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS devices (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		serial_number TEXT NOT NULL,
		location TEXT NOT NULL DEFAULT '',
		tags TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL,
		token_hash TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(tenant_id, serial_number),
		FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_token_hash ON devices(token_hash)`,
	`CREATE TABLE IF NOT EXISTS telemetry_points (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		device_id TEXT NOT NULL,
		metric_name TEXT NOT NULL,
		value REAL NOT NULL,
		timestamp TEXT NOT NULL,
		labels TEXT NOT NULL DEFAULT '{}',
		received_at TEXT NOT NULL,
		idempotency_key TEXT,
		FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_telemetry_idempotency ON telemetry_points(tenant_id, idempotency_key)`,
	`CREATE INDEX IF NOT EXISTS idx_telemetry_query ON telemetry_points(tenant_id, device_id, metric_name, timestamp)`,
	`CREATE TABLE IF NOT EXISTS telemetry_aggregates (
		tenant_id TEXT NOT NULL,
		device_id TEXT NOT NULL,
		metric_name TEXT NOT NULL,
		granularity TEXT NOT NULL,
		bucket_time TEXT NOT NULL,
		count INTEGER NOT NULL,
		sum REAL NOT NULL,
		min REAL NOT NULL,
		max REAL NOT NULL,
		PRIMARY KEY (tenant_id, device_id, metric_name, granularity, bucket_time),
		FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS alert_rules (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		name TEXT NOT NULL,
		device_id TEXT,
		device_tag TEXT NOT NULL DEFAULT '',
		metric_name TEXT NOT NULL,
		operator TEXT NOT NULL,
		threshold REAL NOT NULL,
		window_ms INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL,
		level TEXT NOT NULL,
		cooldown_ms INTEGER NOT NULL,
		channels TEXT NOT NULL,
		enabled INTEGER NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_alert_rules_tenant_enabled ON alert_rules(tenant_id, enabled)`,
	`CREATE TABLE IF NOT EXISTS alert_states (
		tenant_id TEXT NOT NULL,
		rule_id TEXT NOT NULL,
		device_id TEXT NOT NULL,
		metric_name TEXT NOT NULL,
		state TEXT NOT NULL,
		since TEXT NOT NULL DEFAULT '',
		last_value REAL NOT NULL DEFAULT 0,
		last_event_at TEXT NOT NULL DEFAULT '',
		updated_at TEXT NOT NULL,
		PRIMARY KEY (tenant_id, rule_id, device_id, metric_name),
		FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		FOREIGN KEY (rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE,
		FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS alert_events (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		rule_id TEXT NOT NULL,
		rule_name TEXT NOT NULL,
		device_id TEXT NOT NULL,
		metric_name TEXT NOT NULL,
		value REAL NOT NULL,
		event_type TEXT NOT NULL,
		level TEXT NOT NULL,
		channels TEXT NOT NULL DEFAULT '[]',
		message TEXT NOT NULL,
		created_at TEXT NOT NULL,
		FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		FOREIGN KEY (rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE,
		FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
	)`,
	`CREATE INDEX IF NOT EXISTS idx_alert_events_query ON alert_events(tenant_id, created_at)`,
	`CREATE TABLE IF NOT EXISTS notification_attempts (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		event_id TEXT NOT NULL,
		channel TEXT NOT NULL,
		status TEXT NOT NULL,
		attempts INTEGER NOT NULL DEFAULT 0,
		next_attempt_at TEXT NOT NULL,
		last_error TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
		FOREIGN KEY (event_id) REFERENCES alert_events(id) ON DELETE CASCADE
	)`,
	`CREATE INDEX IF NOT EXISTS idx_notification_pending ON notification_attempts(status, next_attempt_at)`,
}

// Migrate creates all required tables for the SQLite development database.
func Migrate(ctx context.Context, db *sql.DB) error {
	for _, statement := range schema {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply sqlite schema: %w", err)
		}
	}
	return nil
}
