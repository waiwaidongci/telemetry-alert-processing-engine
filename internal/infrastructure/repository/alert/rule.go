package alert

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/alert"
	"github.com/example/telemetry-alert/internal/infrastructure/repository/repoutil"
)

type RuleRepository struct {
	db *sql.DB
}

func NewRuleRepository(db *sql.DB) *RuleRepository {
	return &RuleRepository{db: db}
}

func (r *RuleRepository) Create(ctx context.Context, rule alert.Rule) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO alert_rules
		 (id, tenant_id, name, device_id, device_tag, metric_name, operator, threshold,
		  window_ms, duration_ms, level, cooldown_ms, channels, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.TenantID, rule.Name, nullable(rule.DeviceID), rule.DeviceTag, rule.MetricName,
		string(rule.Operator), rule.Threshold, rule.Window.Milliseconds(), rule.Duration.Milliseconds(),
		string(rule.Level), rule.Cooldown.Milliseconds(), repoutil.MarshalStrings(rule.Channels),
		boolInt(rule.Enabled), repoutil.FormatTime(rule.CreatedAt), repoutil.FormatTime(rule.UpdatedAt))
	if err != nil {
		return fmt.Errorf("insert alert rule: %w", err)
	}
	return nil
}

func (r *RuleRepository) Get(ctx context.Context, tenantID, id string) (alert.Rule, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, name, device_id, device_tag, metric_name, operator, threshold,
		 window_ms, duration_ms, level, cooldown_ms, channels, enabled, created_at, updated_at
		 FROM alert_rules WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return scanRule(row)
}

func (r *RuleRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]alert.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, name, device_id, device_tag, metric_name, operator, threshold,
		 window_ms, duration_ms, level, cooldown_ms, channels, enabled, created_at, updated_at
		 FROM alert_rules WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query alert rules: %w", err)
	}
	defer rows.Close()
	return scanRules(rows)
}

func (r *RuleRepository) Update(ctx context.Context, rule alert.Rule) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE alert_rules SET name = ?, device_id = ?, device_tag = ?, metric_name = ?,
		 operator = ?, threshold = ?, window_ms = ?, duration_ms = ?, level = ?,
		 cooldown_ms = ?, channels = ?, enabled = ?, updated_at = ?
		 WHERE tenant_id = ? AND id = ?`,
		rule.Name, nullable(rule.DeviceID), rule.DeviceTag, rule.MetricName, string(rule.Operator),
		rule.Threshold, rule.Window.Milliseconds(), rule.Duration.Milliseconds(), string(rule.Level),
		rule.Cooldown.Milliseconds(), repoutil.MarshalStrings(rule.Channels), boolInt(rule.Enabled),
		repoutil.FormatTime(rule.UpdatedAt), rule.TenantID, rule.ID)
	if err != nil {
		return fmt.Errorf("update alert rule: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (r *RuleRepository) Delete(ctx context.Context, tenantID, id string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM alert_rules WHERE tenant_id = ? AND id = ?`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete alert rule: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (r *RuleRepository) ListEnabled(ctx context.Context, tenantID string) ([]alert.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, name, device_id, device_tag, metric_name, operator, threshold,
		 window_ms, duration_ms, level, cooldown_ms, channels, enabled, created_at, updated_at
		 FROM alert_rules WHERE tenant_id = ? AND enabled = 1 ORDER BY id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query enabled alert rules: %w", err)
	}
	defer rows.Close()
	return scanRules(rows)
}

func (r *RuleRepository) ListEnabledAll(ctx context.Context) ([]alert.Rule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, name, device_id, device_tag, metric_name, operator, threshold,
		 window_ms, duration_ms, level, cooldown_ms, channels, enabled, created_at, updated_at
		 FROM alert_rules WHERE enabled = 1 ORDER BY tenant_id, id`)
	if err != nil {
		return nil, fmt.Errorf("query all enabled alert rules: %w", err)
	}
	defer rows.Close()
	return scanRules(rows)
}

func scanRules(rows *sql.Rows) ([]alert.Rule, error) {
	var out []alert.Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRule(sc scanner) (alert.Rule, error) {
	var rule alert.Rule
	var deviceID sql.NullString
	var channels, created, updated string
	var windowMs, durationMs, cooldownMs int64
	var enabled int
	if err := sc.Scan(&rule.ID, &rule.TenantID, &rule.Name, &deviceID, &rule.DeviceTag,
		&rule.MetricName, &rule.Operator, &rule.Threshold, &windowMs, &durationMs,
		&rule.Level, &cooldownMs, &channels, &enabled, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return alert.Rule{}, application.ErrNotFound
		}
		return alert.Rule{}, fmt.Errorf("scan alert rule: %w", err)
	}
	rule.DeviceID = deviceID.String
	rule.Window = time.Duration(windowMs) * time.Millisecond
	rule.Duration = time.Duration(durationMs) * time.Millisecond
	rule.Cooldown = time.Duration(cooldownMs) * time.Millisecond
	rule.Enabled = enabled == 1
	var err error
	rule.Channels, err = repoutil.UnmarshalStrings(channels)
	if err != nil {
		return alert.Rule{}, err
	}
	rule.CreatedAt, err = repoutil.ParseTime(created)
	if err != nil {
		return alert.Rule{}, err
	}
	rule.UpdatedAt, err = repoutil.ParseTime(updated)
	if err != nil {
		return alert.Rule{}, err
	}
	return rule, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
