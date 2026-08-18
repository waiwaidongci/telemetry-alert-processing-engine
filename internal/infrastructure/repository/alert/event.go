package alert

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/telemetry-alert/internal/application"
	appevent "github.com/example/telemetry-alert/internal/application/event"
	"github.com/example/telemetry-alert/internal/domain/alert"
	"github.com/example/telemetry-alert/internal/infrastructure/repository/repoutil"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event alert.Event) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO alert_events
		 (id, tenant_id, rule_id, rule_name, device_id, metric_name, value, event_type, level, channels, message, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.TenantID, event.RuleID, event.RuleName, event.DeviceID, event.MetricName,
		event.Value, string(event.EventType), string(event.Level), repoutil.MarshalStrings(event.Channels),
		event.Message, repoutil.FormatTime(event.CreatedAt))
	if err != nil {
		return fmt.Errorf("insert alert event: %w", err)
	}
	return nil
}

func (r *EventRepository) Get(ctx context.Context, id string) (alert.Event, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, rule_id, rule_name, device_id, metric_name, value, event_type, level, channels, message, created_at
		 FROM alert_events WHERE id = ?`, id)
	return scanEvent(row)
}

func (r *EventRepository) List(ctx context.Context, tenantID string, filter appevent.Filter) ([]alert.Event, error) {
	query := `SELECT id, tenant_id, rule_id, rule_name, device_id, metric_name, value, event_type, level, channels, message, created_at
		FROM alert_events WHERE tenant_id = ?`
	args := []any{tenantID}
	if filter.DeviceID != "" {
		query += " AND device_id = ?"
		args = append(args, filter.DeviceID)
	}
	if filter.RuleID != "" {
		query += " AND rule_id = ?"
		args = append(args, filter.RuleID)
	}
	if filter.EventType != "" {
		query += " AND event_type = ?"
		args = append(args, string(filter.EventType))
	}
	if !filter.Start.IsZero() {
		query += " AND created_at >= ?"
		args = append(args, repoutil.FormatTime(filter.Start))
	}
	if !filter.End.IsZero() {
		query += " AND created_at < ?"
		args = append(args, repoutil.FormatTime(filter.End))
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query alert events: %w", err)
	}
	defer rows.Close()
	var out []alert.Event
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

type eventScanner interface {
	Scan(dest ...any) error
}

func scanEvent(sc eventScanner) (alert.Event, error) {
	var e alert.Event
	var channels, created string
	if err := sc.Scan(&e.ID, &e.TenantID, &e.RuleID, &e.RuleName, &e.DeviceID, &e.MetricName,
		&e.Value, &e.EventType, &e.Level, &channels, &e.Message, &created); err != nil {
		if err == sql.ErrNoRows {
			return alert.Event{}, application.ErrNotFound
		}
		return alert.Event{}, fmt.Errorf("scan alert event: %w", err)
	}
	var err error
	e.Channels, err = repoutil.UnmarshalStrings(channels)
	if err != nil {
		return alert.Event{}, err
	}
	e.CreatedAt, err = repoutil.ParseTime(created)
	if err != nil {
		return alert.Event{}, err
	}
	return e, nil
}
