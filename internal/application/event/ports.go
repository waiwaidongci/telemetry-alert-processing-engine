package event

import (
	"context"
	"time"

	"github.com/example/telemetry-alert/internal/domain/alert"
)

type Repository interface {
	Create(ctx context.Context, event alert.Event) error
	Get(ctx context.Context, id string) (alert.Event, error)
	List(ctx context.Context, tenantID string, filter Filter) ([]alert.Event, error)
}

type Filter struct {
	DeviceID  string
	RuleID    string
	EventType alert.EventType
	Start     time.Time
	End       time.Time
	Limit     int
	Offset    int
}
