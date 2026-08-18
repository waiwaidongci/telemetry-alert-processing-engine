package device

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/device"
	"github.com/example/telemetry-alert/internal/infrastructure/repository/repoutil"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, d device.Device) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO devices (id, tenant_id, name, type, serial_number, location, tags, status, token_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.TenantID, d.Name, d.Type, d.SerialNumber, d.Location,
		repoutil.MarshalMap(d.Tags), string(d.Status), d.TokenHash,
		repoutil.FormatTime(d.CreatedAt), repoutil.FormatTime(d.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id string) (device.Device, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, name, type, serial_number, location, tags, status, token_hash, created_at, updated_at
		 FROM devices WHERE tenant_id = ? AND id = ?`, tenantID, id)
	return scanDevice(row)
}

func (r *Repository) GetByTokenHash(ctx context.Context, hash string) (device.Device, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, name, type, serial_number, location, tags, status, token_hash, created_at, updated_at
		 FROM devices WHERE token_hash = ?`, hash)
	return scanDevice(row)
}

func (r *Repository) List(ctx context.Context, tenantID, status string, limit, offset int) ([]device.Device, error) {
	query := `SELECT id, tenant_id, name, type, serial_number, location, tags, status, token_hash, created_at, updated_at
		FROM devices WHERE tenant_id = ?`
	args := []any{tenantID}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()
	var out []device.Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, d device.Device) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE devices SET name = ?, type = ?, serial_number = ?, location = ?, tags = ?, status = ?, updated_at = ?
		 WHERE tenant_id = ? AND id = ?`,
		d.Name, d.Type, d.SerialNumber, d.Location, repoutil.MarshalMap(d.Tags), string(d.Status),
		repoutil.FormatTime(d.UpdatedAt), d.TenantID, d.ID)
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, id string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM devices WHERE tenant_id = ? AND id = ?`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return application.ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDevice(sc rowScanner) (device.Device, error) {
	var d device.Device
	var tags, created, updated string
	var status string
	if err := sc.Scan(&d.ID, &d.TenantID, &d.Name, &d.Type, &d.SerialNumber, &d.Location,
		&tags, &status, &d.TokenHash, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return device.Device{}, application.ErrNotFound
		}
		return device.Device{}, fmt.Errorf("scan device: %w", err)
	}
	var err error
	d.Tags, err = repoutil.UnmarshalMap(tags)
	if err != nil {
		return device.Device{}, err
	}
	d.Status = device.Status(strings.TrimSpace(status))
	d.CreatedAt, err = repoutil.ParseTime(created)
	if err != nil {
		return device.Device{}, err
	}
	d.UpdatedAt, err = repoutil.ParseTime(updated)
	if err != nil {
		return device.Device{}, err
	}
	return d, nil
}
