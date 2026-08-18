package device

import (
	"context"
	"fmt"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/device"
	"github.com/example/telemetry-alert/internal/system"
)

type Service struct {
	repo   Repository
	ids    system.IDGenerator
	clock  system.Clock
	tokens system.TokenHasher
}

type CreatedDevice struct {
	Device device.Device
	Token  string
}

func NewService(repo Repository, ids system.IDGenerator, clock system.Clock, tokens system.TokenHasher) *Service {
	return &Service{repo: repo, ids: ids, clock: clock, tokens: tokens}
}

func (s *Service) Create(ctx context.Context, tenantID string, in CreateInput) (CreatedDevice, error) {
	if tenantID == "" {
		return CreatedDevice{}, application.ValidationError{Field: "tenant_id", Message: "must not be empty"}
	}
	if in.Name == "" {
		return CreatedDevice{}, application.ValidationError{Field: "name", Message: "must not be empty"}
	}
	if in.Type == "" {
		return CreatedDevice{}, application.ValidationError{Field: "type", Message: "must not be empty"}
	}
	if in.SerialNumber == "" {
		return CreatedDevice{}, application.ValidationError{Field: "serial_number", Message: "must not be empty"}
	}
	token, hash, err := s.tokens.Generate()
	if err != nil {
		return CreatedDevice{}, fmt.Errorf("generate device token: %w", err)
	}
	now := s.clock.Now().UTC()
	d := device.Device{
		ID:           s.ids.New(),
		TenantID:     tenantID,
		Name:         in.Name,
		Type:         in.Type,
		SerialNumber: in.SerialNumber,
		Location:     in.Location,
		Tags:         cloneStringMap(in.Tags),
		Status:       device.StatusActive,
		TokenHash:    hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return CreatedDevice{}, fmt.Errorf("create device: %w", err)
	}
	return CreatedDevice{Device: d, Token: token}, nil
}

func (s *Service) Get(ctx context.Context, tenantID, id string) (device.Device, error) {
	if tenantID == "" || id == "" {
		return device.Device{}, application.ValidationError{Field: "id", Message: "tenant and device id are required"}
	}
	d, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return device.Device{}, fmt.Errorf("get device: %w", err)
	}
	return d, nil
}

func (s *Service) List(ctx context.Context, tenantID, status string, limit, offset int) ([]device.Device, error) {
	if tenantID == "" {
		return nil, application.ValidationError{Field: "tenant_id", Message: "must not be empty"}
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	list, err := s.repo.List(ctx, tenantID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	return list, nil
}

func (s *Service) Update(ctx context.Context, tenantID, id string, in CreateInput) (device.Device, error) {
	if in.Name == "" || in.Type == "" || in.SerialNumber == "" {
		return device.Device{}, application.ValidationError{Field: "body", Message: "name, type, and serial_number are required"}
	}
	d, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return device.Device{}, fmt.Errorf("get device for update: %w", err)
	}
	d.Name = in.Name
	d.Type = in.Type
	d.SerialNumber = in.SerialNumber
	d.Location = in.Location
	d.Tags = cloneStringMap(in.Tags)
	d.UpdatedAt = s.clock.Now().UTC()
	if err := s.repo.Update(ctx, d); err != nil {
		return device.Device{}, fmt.Errorf("update device: %w", err)
	}
	return d, nil
}

func (s *Service) SetStatus(ctx context.Context, tenantID, id string, status device.Status) (device.Device, error) {
	if status != device.StatusActive && status != device.StatusInactive {
		return device.Device{}, application.ValidationError{Field: "status", Message: "must be active or inactive"}
	}
	d, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return device.Device{}, fmt.Errorf("get device for status: %w", err)
	}
	d.Status = status
	d.UpdatedAt = s.clock.Now().UTC()
	if err := s.repo.Update(ctx, d); err != nil {
		return device.Device{}, fmt.Errorf("update device status: %w", err)
	}
	return d, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, id string) error {
	if tenantID == "" || id == "" {
		return application.ValidationError{Field: "id", Message: "tenant and device id are required"}
	}
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	return nil
}

// Authenticate verifies a raw access token and returns the matching device.
// The returned hash is intentionally the same value stored in the database.
func (s *Service) Authenticate(ctx context.Context, token string) (device.Device, error) {
	if token == "" {
		return device.Device{}, application.ErrUnauthorized
	}
	hash := s.tokens.Hash(token)
	d, err := s.repo.GetByTokenHash(ctx, hash)
	if err != nil {
		return device.Device{}, fmt.Errorf("authenticate device: %w", err)
	}
	if !d.IsActive() {
		return device.Device{}, application.ErrDeviceInactive
	}
	return d, nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
