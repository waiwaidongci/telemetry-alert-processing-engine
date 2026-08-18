package device

import (
	"context"

	"github.com/example/telemetry-alert/internal/domain/device"
)

type Repository interface {
	Create(ctx context.Context, d device.Device) error
	Get(ctx context.Context, tenantID, id string) (device.Device, error)
	GetByTokenHash(ctx context.Context, hash string) (device.Device, error)
	List(ctx context.Context, tenantID string, status string, limit, offset int) ([]device.Device, error)
	Update(ctx context.Context, d device.Device) error
	Delete(ctx context.Context, tenantID, id string) error
}

// CreateInput is the validated request payload for a new device.
type CreateInput struct {
	Name         string
	Type         string
	SerialNumber string
	Location     string
	Tags         map[string]string
}
