package alertrule

import (
	"context"
	"time"

	"github.com/example/telemetry-alert/internal/domain/alert"
	"github.com/example/telemetry-alert/internal/domain/device"
)

type RuleRepository interface {
	Create(ctx context.Context, rule alert.Rule) error
	Get(ctx context.Context, tenantID, id string) (alert.Rule, error)
	List(ctx context.Context, tenantID string, limit, offset int) ([]alert.Rule, error)
	Update(ctx context.Context, rule alert.Rule) error
	Delete(ctx context.Context, tenantID, id string) error
	ListEnabled(ctx context.Context, tenantID string) ([]alert.Rule, error)
	ListEnabledAll(ctx context.Context) ([]alert.Rule, error)
}

type StateRepository interface {
	Get(ctx context.Context, tenantID, ruleID, deviceID, metricName string) (alert.Instance, error)
	Upsert(ctx context.Context, state alert.Instance) error
	DeleteByRule(ctx context.Context, tenantID, ruleID string) error
	DeleteByDevice(ctx context.Context, tenantID, deviceID string) error
}

type EventWriter interface {
	Create(ctx context.Context, event alert.Event) error
}

type Dispatcher interface {
	Dispatch(ctx context.Context, event alert.Event)
}

type DeviceReader interface {
	Get(ctx context.Context, tenantID, id string) (device.Device, error)
	List(ctx context.Context, tenantID, status string, limit, offset int) ([]device.Device, error)
}

type MetricWindowReader interface {
	ReadWindow(ctx context.Context, tenantID, deviceID, metricName string, since time.Time) (MetricSample, error)
}

type MetricSample struct {
	Has   bool
	Value float64
	Avg   float64
	Count int
}
