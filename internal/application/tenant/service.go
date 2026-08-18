package tenant

import (
	"context"
	"fmt"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/tenant"
	"github.com/example/telemetry-alert/internal/system"
)

type Service struct {
	repo  Repository
	ids   system.IDGenerator
	clock system.Clock
}

func NewService(repo Repository, ids system.IDGenerator, clock system.Clock) *Service {
	return &Service{repo: repo, ids: ids, clock: clock}
}

func (s *Service) Create(ctx context.Context, name string) (tenant.Tenant, error) {
	if name == "" {
		return tenant.Tenant{}, application.ValidationError{Field: "name", Message: "must not be empty"}
	}
	now := s.clock.Now().UTC()
	t := tenant.Tenant{ID: s.ids.New(), Name: name, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(ctx, t); err != nil {
		return tenant.Tenant{}, fmt.Errorf("create tenant: %w", err)
	}
	return t, nil
}

func (s *Service) Get(ctx context.Context, id string) (tenant.Tenant, error) {
	if id == "" {
		return tenant.Tenant{}, application.ValidationError{Field: "id", Message: "must not be empty"}
	}
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return tenant.Tenant{}, fmt.Errorf("get tenant: %w", err)
	}
	return t, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]tenant.Tenant, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	list, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	return list, nil
}

func (s *Service) Update(ctx context.Context, id, name string) (tenant.Tenant, error) {
	if id == "" {
		return tenant.Tenant{}, application.ValidationError{Field: "id", Message: "must not be empty"}
	}
	if name == "" {
		return tenant.Tenant{}, application.ValidationError{Field: "name", Message: "must not be empty"}
	}
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return tenant.Tenant{}, fmt.Errorf("get tenant for update: %w", err)
	}
	t.Name = name
	t.UpdatedAt = s.clock.Now().UTC()
	if err := s.repo.Update(ctx, t); err != nil {
		return tenant.Tenant{}, fmt.Errorf("update tenant: %w", err)
	}
	return t, nil
}
