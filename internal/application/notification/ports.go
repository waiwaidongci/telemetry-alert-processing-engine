package notification

import (
	"context"
	"time"

	"github.com/example/telemetry-alert/internal/domain/alert"
	"github.com/example/telemetry-alert/internal/domain/notification"
)

type Repository interface {
	Create(ctx context.Context, attempt notification.Attempt) error
	Get(ctx context.Context, id string) (notification.Attempt, error)
	Update(ctx context.Context, attempt notification.Attempt) error
	ListPending(ctx context.Context, before time.Time, limit int) ([]notification.Attempt, error)
	ListFailures(ctx context.Context, tenantID string, limit, offset int) ([]notification.Attempt, error)
	DeleteOlder(ctx context.Context, before time.Time, batchSize int) (int64, error)
}

type Sender interface {
	Send(ctx context.Context, event alert.Event, channel notification.Channel) error
}

type EventLoader interface {
	Get(ctx context.Context, id string) (alert.Event, error)
}

type Config struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	RequestTimeout time.Duration
	WorkerCount    int
}
