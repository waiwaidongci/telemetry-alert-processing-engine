package notification

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/notification"
	"github.com/example/telemetry-alert/internal/infrastructure/repository/repoutil"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, attempt notification.Attempt) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notification_attempts
		 (id, tenant_id, event_id, channel, status, attempts, next_attempt_at, last_error, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		attempt.ID, attempt.TenantID, attempt.EventID, string(attempt.Channel), string(attempt.Status),
		attempt.Attempts, repoutil.FormatTime(attempt.NextAttemptAt), attempt.LastError,
		repoutil.FormatTime(attempt.CreatedAt), repoutil.FormatTime(attempt.UpdatedAt))
	if err != nil {
		return fmt.Errorf("insert notification attempt: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id string) (notification.Attempt, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, event_id, channel, status, attempts, next_attempt_at, last_error, created_at, updated_at
		 FROM notification_attempts WHERE id = ?`, id)
	return scanAttempt(row)
}

func (r *Repository) Update(ctx context.Context, attempt notification.Attempt) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE notification_attempts SET status = ?, attempts = ?, next_attempt_at = ?, last_error = ?, updated_at = ?
		 WHERE id = ?`,
		string(attempt.Status), attempt.Attempts, repoutil.FormatTime(attempt.NextAttemptAt),
		attempt.LastError, repoutil.FormatTime(attempt.UpdatedAt), attempt.ID)
	if err != nil {
		return fmt.Errorf("update notification attempt: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (r *Repository) ListPending(ctx context.Context, before time.Time, limit int) ([]notification.Attempt, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, event_id, channel, status, attempts, next_attempt_at, last_error, created_at, updated_at
		 FROM notification_attempts
		 WHERE status = ? AND next_attempt_at <= ?
		 ORDER BY next_attempt_at ASC LIMIT ?`,
		string(notification.StatusPending), repoutil.FormatTime(before), limit)
	if err != nil {
		return nil, fmt.Errorf("query pending notification attempts: %w", err)
	}
	defer rows.Close()
	return scanAttempts(rows)
}

func (r *Repository) ListFailures(ctx context.Context, tenantID string, limit, offset int) ([]notification.Attempt, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, event_id, channel, status, attempts, next_attempt_at, last_error, created_at, updated_at
		 FROM notification_attempts
		 WHERE tenant_id = ? AND status = ?
		 ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		tenantID, string(notification.StatusFailed), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed notification attempts: %w", err)
	}
	defer rows.Close()
	return scanAttempts(rows)
}

func (r *Repository) DeleteOlder(ctx context.Context, before time.Time, batchSize int) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM notification_attempts WHERE id IN (
			SELECT id FROM notification_attempts WHERE updated_at < ? ORDER BY updated_at LIMIT ?
		)`,
		repoutil.FormatTime(before), batchSize)
	if err != nil {
		return 0, fmt.Errorf("delete old notification attempts: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func scanAttempts(rows *sql.Rows) ([]notification.Attempt, error) {
	var out []notification.Attempt
	for rows.Next() {
		attempt, err := scanAttempt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, attempt)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAttempt(sc scanner) (notification.Attempt, error) {
	var a notification.Attempt
	var next, created, updated string
	if err := sc.Scan(&a.ID, &a.TenantID, &a.EventID, &a.Channel, &a.Status, &a.Attempts,
		&next, &a.LastError, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return notification.Attempt{}, application.ErrNotFound
		}
		return notification.Attempt{}, fmt.Errorf("scan notification attempt: %w", err)
	}
	var err error
	a.NextAttemptAt, err = repoutil.ParseTime(next)
	if err != nil {
		return notification.Attempt{}, err
	}
	a.CreatedAt, err = repoutil.ParseTime(created)
	if err != nil {
		return notification.Attempt{}, err
	}
	a.UpdatedAt, err = repoutil.ParseTime(updated)
	if err != nil {
		return notification.Attempt{}, err
	}
	return a, nil
}
