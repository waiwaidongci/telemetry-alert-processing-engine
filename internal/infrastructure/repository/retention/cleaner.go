package retention

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/example/telemetry-alert/internal/domain/retention"
	"github.com/example/telemetry-alert/internal/infrastructure/repository/repoutil"
)

type Cleaner struct {
	db *sql.DB
}

func NewCleaner(db *sql.DB) *Cleaner {
	return &Cleaner{db: db}
}

func (c *Cleaner) Clean(ctx context.Context, plan retention.Plan) (retention.Result, error) {
	var result retention.Result
	var err error
	if result.RawDeleted, err = deleteBatch(ctx, c.db,
		`DELETE FROM telemetry_points WHERE id IN (
			SELECT id FROM telemetry_points WHERE timestamp < ? ORDER BY timestamp LIMIT ?
		)`, plan.RawBefore, plan.BatchSize); err != nil {
		return result, fmt.Errorf("delete raw telemetry: %w", err)
	}
	if result.AggregateDeleted, err = deleteBatch(ctx, c.db,
		`DELETE FROM telemetry_aggregates WHERE rowid IN (
			SELECT rowid FROM telemetry_aggregates WHERE bucket_time < ? ORDER BY bucket_time LIMIT ?
		)`, plan.AggregateBefore, plan.BatchSize); err != nil {
		return result, fmt.Errorf("delete aggregate telemetry: %w", err)
	}
	if result.EventsDeleted, err = deleteBatch(ctx, c.db,
		`DELETE FROM alert_events WHERE id IN (
			SELECT id FROM alert_events WHERE created_at < ? ORDER BY created_at LIMIT ?
		)`, plan.EventsBefore, plan.BatchSize); err != nil {
		return result, fmt.Errorf("delete alert events: %w", err)
	}
	if result.FailuresDeleted, err = deleteBatch(ctx, c.db,
		`DELETE FROM notification_attempts WHERE id IN (
			SELECT id FROM notification_attempts WHERE updated_at < ? ORDER BY updated_at LIMIT ?
		)`, plan.FailuresBefore, plan.BatchSize); err != nil {
		return result, fmt.Errorf("delete notification failures: %w", err)
	}
	return result, nil
}

func deleteBatch(ctx context.Context, db *sql.DB, query string, before time.Time, limit int) (int64, error) {
	result, err := db.ExecContext(ctx, query, repoutil.FormatTime(before), limit)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, nil
}
