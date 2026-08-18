package tenant

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/tenant"
	"github.com/example/telemetry-alert/internal/infrastructure/repository/repoutil"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, t tenant.Tenant) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenants (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		t.ID, t.Name, repoutil.FormatTime(t.CreatedAt), repoutil.FormatTime(t.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id string) (tenant.Tenant, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, created_at, updated_at FROM tenants WHERE id = ?`, id)
	var t tenant.Tenant
	var created, updated string
	if err := row.Scan(&t.ID, &t.Name, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return tenant.Tenant{}, application.ErrNotFound
		}
		return tenant.Tenant{}, fmt.Errorf("scan tenant: %w", err)
	}
	var err error
	t.CreatedAt, err = repoutil.ParseTime(created)
	if err != nil {
		return tenant.Tenant{}, err
	}
	t.UpdatedAt, err = repoutil.ParseTime(updated)
	if err != nil {
		return tenant.Tenant{}, err
	}
	return t, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]tenant.Tenant, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, created_at, updated_at FROM tenants ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	defer rows.Close()
	var out []tenant.Tenant
	for rows.Next() {
		var t tenant.Tenant
		var created, updated string
		if err := rows.Scan(&t.ID, &t.Name, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		t.CreatedAt, err = repoutil.ParseTime(created)
		if err != nil {
			return nil, err
		}
		t.UpdatedAt, err = repoutil.ParseTime(updated)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, t tenant.Tenant) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET name = ?, updated_at = ? WHERE id = ?`,
		t.Name, repoutil.FormatTime(t.UpdatedAt), t.ID)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return application.ErrNotFound
	}
	return nil
}
