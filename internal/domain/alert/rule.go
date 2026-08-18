package alert

import "time"

type Operator string

const (
	OpGreaterThan        Operator = "gt"
	OpGreaterThanOrEqual Operator = "gte"
	OpLessThan           Operator = "lt"
	OpLessThanOrEqual    Operator = "lte"
	OpEqual              Operator = "eq"
)

type Level string

const (
	LevelInfo     Level = "info"
	LevelWarning  Level = "warning"
	LevelCritical Level = "critical"
)

// Rule is a user-defined condition evaluated over telemetry.
type Rule struct {
	ID         string        `json:"id"`
	TenantID   string        `json:"tenant_id"`
	Name       string        `json:"name"`
	DeviceID   string        `json:"device_id,omitempty"`
	DeviceTag  string        `json:"device_tag,omitempty"`
	MetricName string        `json:"metric_name"`
	Operator   Operator      `json:"operator"`
	Threshold  float64       `json:"threshold"`
	Window     time.Duration `json:"window"`
	Duration   time.Duration `json:"duration"`
	Level      Level         `json:"level"`
	Cooldown   time.Duration `json:"cooldown"`
	Channels   []string      `json:"channels"`
	Enabled    bool          `json:"enabled"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

func (r Rule) AllowedChannel(name string) bool {
	for _, c := range r.Channels {
		if c == name {
			return true
		}
	}
	return false
}
