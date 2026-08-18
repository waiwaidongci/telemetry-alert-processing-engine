package device

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

// Device represents a physical or logical data source owned by a tenant.
type Device struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	SerialNumber string            `json:"serial_number"`
	Location     string            `json:"location"`
	Tags         map[string]string `json:"tags,omitempty"`
	Status       Status            `json:"status"`
	TokenHash    string            `json:"-"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func (d Device) IsActive() bool { return d.Status == StatusActive }

func (d Device) HasTag(key, value string) bool {
	return d.Tags[key] == value
}
