package alert

import "time"

type State string

const (
	StateNormal    State = "normal"
	StatePending   State = "pending"
	StateTriggered State = "triggered"
	StateRecovered State = "recovered"
)

// Instance identifies a rule being evaluated for one device/metric pair.
type Instance struct {
	TenantID    string
	RuleID      string
	DeviceID    string
	MetricName  string
	State       State
	Since       time.Time
	LastValue   float64
	LastEventAt time.Time
	UpdatedAt   time.Time
}
