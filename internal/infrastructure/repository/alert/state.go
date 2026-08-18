package alert

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/alert"
	"github.com/example/telemetry-alert/internal/infrastructure/repository/repoutil"
)

type StateRepository struct {
	db *sql.DB
}

func NewStateRepository(db *sql.DB) *StateRepository {
	return &StateRepository{db: db}
}

func (r *StateRepository) Get(ctx context.Context, tenantID, ruleID, deviceID, metricName string) (alert.Instance, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT tenant_id, rule_id, device_id, metric_name, state, since, last_value, last_event_at, updated_at
		 FROM alert_states WHERE tenant_id = ? AND rule_id = ? AND device_id = ? AND metric_name = ?`,
		tenantID, ruleID, deviceID, metricName)
	var state alert.Instance
	var since, lastEvent, updated string
	if err := row.Scan(&state.TenantID, &state.RuleID, &state.DeviceID, &state.MetricName,
		&state.State, &since, &state.LastValue, &lastEvent, &updated); err != nil {
		if err == sql.ErrNoRows {
			return alert.Instance{}, application.ErrNotFound
		}
		return alert.Instance{}, fmt.Errorf("scan alert state: %w", err)
	}
	var err error
	state.Since, err = repoutil.ParseTime(since)
	if err != nil {
		return alert.Instance{}, err
	}
	state.LastEventAt, err = repoutil.ParseTime(lastEvent)
	if err != nil {
		return alert.Instance{}, err
	}
	state.UpdatedAt, err = repoutil.ParseTime(updated)
	if err != nil {
		return alert.Instance{}, err
	}
	return state, nil
}

func (r *StateRepository) Upsert(ctx context.Context, state alert.Instance) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO alert_states
		 (tenant_id, rule_id, device_id, metric_name, state, since, last_value, last_event_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(tenant_id, rule_id, device_id, metric_name)
		 DO UPDATE SET state = excluded.state, since = excluded.since,
		   last_value = excluded.last_value, last_event_at = excluded.last_event_at,
		   updated_at = excluded.updated_at`,
		state.TenantID, state.RuleID, state.DeviceID, state.MetricName, string(state.State),
		repoutil.FormatTime(state.Since), state.LastValue, repoutil.FormatTime(state.LastEventAt),
		repoutil.FormatTime(state.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert alert state: %w", err)
	}
	return nil
}

func (r *StateRepository) DeleteByRule(ctx context.Context, tenantID, ruleID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM alert_states WHERE tenant_id = ? AND rule_id = ?`, tenantID, ruleID)
	if err != nil {
		return fmt.Errorf("delete alert states by rule: %w", err)
	}
	return nil
}

func (r *StateRepository) DeleteByDevice(ctx context.Context, tenantID, deviceID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM alert_states WHERE tenant_id = ? AND device_id = ?`, tenantID, deviceID)
	if err != nil {
		return fmt.Errorf("delete alert states by device: %w", err)
	}
	return nil
}
