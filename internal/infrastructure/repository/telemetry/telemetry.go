package telemetry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	apptelemetry "github.com/example/telemetry-alert/internal/application/telemetry"
	"github.com/example/telemetry-alert/internal/domain/telemetry"
	"github.com/example/telemetry-alert/internal/infrastructure/repository/repoutil"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) InsertBatch(ctx context.Context, points []telemetry.Point) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin telemetry insert: %w", err)
	}
	defer tx.Rollback()
	inserted := 0
	for _, p := range points {
		ok, err := insertPoint(ctx, tx, p)
		if err != nil {
			return 0, err
		}
		if ok {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit telemetry insert: %w", err)
	}
	return inserted, nil
}

func insertPoint(ctx context.Context, tx *sql.Tx, p telemetry.Point) (bool, error) {
	if p.IdempotencyKey != "" {
		var exists int
		if err := tx.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM telemetry_points WHERE tenant_id = ? AND idempotency_key = ?)`,
			p.TenantID, p.IdempotencyKey).Scan(&exists); err != nil {
			return false, fmt.Errorf("check idempotency key in transaction: %w", err)
		}
		if exists == 1 {
			return false, nil
		}
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO telemetry_points (id, tenant_id, device_id, metric_name, value, timestamp, labels, received_at, idempotency_key)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.TenantID, p.DeviceID, p.MetricName, p.Value, repoutil.FormatTime(p.Timestamp),
		repoutil.MarshalMap(p.Labels), repoutil.FormatTime(p.ReceivedAt), nullableString(p.IdempotencyKey))
	if err != nil {
		return false, fmt.Errorf("insert telemetry point: %w", err)
	}
	if err := upsertAggregates(ctx, tx, p); err != nil {
		return false, err
	}
	return true, nil
}

func upsertAggregates(ctx context.Context, tx *sql.Tx, p telemetry.Point) error {
	granularities := []struct {
		name string
		d    time.Duration
	}{
		{string(telemetry.Granularity1Minute), time.Minute},
		{string(telemetry.Granularity5Minute), 5 * time.Minute},
		{string(telemetry.Granularity1Hour), time.Hour},
		{string(telemetry.Granularity1Day), 24 * time.Hour},
	}
	for _, g := range granularities {
		bucket := p.Timestamp.UTC().Truncate(g.d)
		_, err := tx.ExecContext(ctx,
			`INSERT INTO telemetry_aggregates
			 (tenant_id, device_id, metric_name, granularity, bucket_time, count, sum, min, max)
			 VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?)
			 ON CONFLICT(tenant_id, device_id, metric_name, granularity, bucket_time)
			 DO UPDATE SET count = count + 1,
			   sum = sum + excluded.sum,
			   min = MIN(telemetry_aggregates.min, excluded.min),
			   max = MAX(telemetry_aggregates.max, excluded.max)`,
			p.TenantID, p.DeviceID, p.MetricName, g.name, repoutil.FormatTime(bucket),
			p.Value, p.Value, p.Value)
		if err != nil {
			return fmt.Errorf("upsert telemetry aggregate %s: %w", g.name, err)
		}
	}
	return nil
}

func (r *Repository) KeyExists(ctx context.Context, tenantID, key string) (bool, error) {
	if key == "" {
		return false, nil
	}
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM telemetry_points WHERE tenant_id = ? AND idempotency_key = ?)`,
		tenantID, key).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check idempotency key: %w", err)
	}
	return exists == 1, nil
}

func (r *Repository) ListRaw(ctx context.Context, q apptelemetry.RawQuery) ([]telemetry.Point, error) {
	query := `SELECT id, tenant_id, device_id, metric_name, value, timestamp, labels, received_at, idempotency_key
		FROM telemetry_points WHERE tenant_id = ? AND timestamp >= ? AND timestamp < ?`
	args := []any{q.TenantID, repoutil.FormatTime(q.Start), repoutil.FormatTime(q.End)}
	if q.DeviceID != "" {
		query += " AND device_id = ?"
		args = append(args, q.DeviceID)
	}
	if q.MetricName != "" {
		query += " AND metric_name = ?"
		args = append(args, q.MetricName)
	}
	query += " ORDER BY timestamp ASC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query raw telemetry: %w", err)
	}
	defer rows.Close()
	var out []telemetry.Point
	for rows.Next() {
		var p telemetry.Point
		var labels, timestamp, received, idempotency sql.NullString
		if err := rows.Scan(&p.ID, &p.TenantID, &p.DeviceID, &p.MetricName, &p.Value,
			&timestamp, &labels, &received, &idempotency); err != nil {
			return nil, fmt.Errorf("scan raw telemetry: %w", err)
		}
		p.Timestamp, err = repoutil.ParseTime(timestamp.String)
		if err != nil {
			return nil, err
		}
		p.ReceivedAt, err = repoutil.ParseTime(received.String)
		if err != nil {
			return nil, err
		}
		p.Labels, err = repoutil.UnmarshalMap(labels.String)
		if err != nil {
			return nil, err
		}
		p.IdempotencyKey = idempotency.String
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) ListAggregates(ctx context.Context, q apptelemetry.AggregateQuery) ([]telemetry.AggregateBucket, error) {
	query := `SELECT tenant_id, device_id, metric_name, granularity, bucket_time, count, sum, min, max
		FROM telemetry_aggregates WHERE tenant_id = ? AND granularity = ? AND bucket_time >= ? AND bucket_time < ?`
	args := []any{q.TenantID, string(q.Granularity), repoutil.FormatTime(q.Start), repoutil.FormatTime(q.End)}
	if q.DeviceID != "" {
		query += " AND device_id = ?"
		args = append(args, q.DeviceID)
	}
	if q.MetricName != "" {
		query += " AND metric_name = ?"
		args = append(args, q.MetricName)
	}
	query += " ORDER BY bucket_time ASC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query aggregate telemetry: %w", err)
	}
	defer rows.Close()
	var out []telemetry.AggregateBucket
	for rows.Next() {
		var b telemetry.AggregateBucket
		var bucket string
		if err := rows.Scan(&b.TenantID, &b.DeviceID, &b.MetricName, &b.Granularity, &bucket,
			&b.Count, &b.Sum, &b.Min, &b.Max); err != nil {
			return nil, fmt.Errorf("scan aggregate telemetry: %w", err)
		}
		b.BucketTime, err = repoutil.ParseTime(bucket)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) WindowStats(ctx context.Context, q apptelemetry.WindowStatsQuery) (apptelemetry.WindowStats, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(value), 0), COALESCE(MIN(value), 0), COALESCE(MAX(value), 0),
		 COALESCE((SELECT value FROM telemetry_points
		   WHERE tenant_id = ? AND device_id = ? AND metric_name = ? AND timestamp >= ? AND timestamp < ?
		   ORDER BY timestamp DESC LIMIT 1), 0)
		 FROM telemetry_points
		 WHERE tenant_id = ? AND device_id = ? AND metric_name = ? AND timestamp >= ? AND timestamp < ?`,
		q.TenantID, q.DeviceID, q.MetricName, repoutil.FormatTime(q.Since), repoutil.FormatTime(q.Until),
		q.TenantID, q.DeviceID, q.MetricName, repoutil.FormatTime(q.Since), repoutil.FormatTime(q.Until))
	var count int
	var sum, min, max, last float64
	if err := row.Scan(&count, &sum, &min, &max, &last); err != nil {
		return apptelemetry.WindowStats{}, fmt.Errorf("scan telemetry window stats: %w", err)
	}
	return apptelemetry.WindowStats{
		Has:   count > 0,
		Count: count,
		Sum:   sum,
		Min:   min,
		Max:   max,
		Last:  last,
	}, nil
}

func nullableString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
