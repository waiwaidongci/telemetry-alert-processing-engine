package event

import (
	"context"
	"fmt"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/alert"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, event alert.Event) error {
	if event.ID == "" || event.TenantID == "" {
		return application.ValidationError{Field: "event", Message: "id and tenant_id are required"}
	}
	if err := s.repo.Create(ctx, event); err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id string) (alert.Event, error) {
	if id == "" {
		return alert.Event{}, application.ValidationError{Field: "id", Message: "must not be empty"}
	}
	event, err := s.repo.Get(ctx, id)
	if err != nil {
		return alert.Event{}, fmt.Errorf("get event: %w", err)
	}
	return event, nil
}

func (s *Service) List(ctx context.Context, tenantID string, filter Filter) ([]alert.Event, error) {
	if tenantID == "" {
		return nil, application.ValidationError{Field: "tenant_id", Message: "must not be empty"}
	}
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	list, err := s.repo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return list, nil
}
