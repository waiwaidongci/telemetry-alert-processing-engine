package alert

import "time"

type EventType string

const (
	EventTriggered  EventType = "triggered"
	EventContinuing EventType = "continuing"
	EventRecovered  EventType = "recovered"
)

// Event is an immutable record of a state transition or continued alarm.
type Event struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	RuleID     string    `json:"rule_id"`
	RuleName   string    `json:"rule_name"`
	DeviceID   string    `json:"device_id"`
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	EventType  EventType `json:"event_type"`
	Level      Level     `json:"level"`
	Channels   []string  `json:"channels,omitempty"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}
