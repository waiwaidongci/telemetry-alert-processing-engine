package tenant

import (
	"context"

	"github.com/example/telemetry-alert/internal/domain/tenant"
)

type Repository interface {
	Create(ctx context.Context, t tenant.Tenant) error
	Get(ctx context.Context, id string) (tenant.Tenant, error)
	List(ctx context.Context, limit, offset int) ([]tenant.Tenant, error)
	Update(ctx context.Context, t tenant.Tenant) error
}
