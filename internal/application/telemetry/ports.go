package telemetry

import (
	"context"
	"time"

	"github.com/example/telemetry-alert/internal/domain/telemetry"
)

// IngestRepository persists raw samples and their idempotency keys.
type IngestRepository interface {
	InsertBatch(ctx context.Context, points []telemetry.Point) (inserted int, err error)
	KeyExists(ctx context.Context, tenantID, key string) (bool, error)
}

// QueryRepository reads raw samples and aggregate buckets.
type QueryRepository interface {
	ListRaw(ctx context.Context, q RawQuery) ([]telemetry.Point, error)
	ListAggregates(ctx context.Context, q AggregateQuery) ([]telemetry.AggregateBucket, error)
	WindowStats(ctx context.Context, q WindowStatsQuery) (WindowStats, error)
}

// Evaluator is called after each successful ingest so alarms can be evaluated
// incrementally. The implementation must not block ingest for long.
type Evaluator interface {
	EvaluateDevice(ctx context.Context, tenantID, deviceID string)
}

// RateLimiter allows ingest throttling per tenant and device.
type RateLimiter interface {
	Allow(ctx context.Context, key string, burst int) bool
}

type IngestInput struct {
	DeviceID       string
	MetricName     string
	Value          float64
	Timestamp      time.Time
	Labels         map[string]string
	IdempotencyKey string
}

type RawQuery struct {
	TenantID   string
	DeviceID   string
	MetricName string
	Start      time.Time
	End        time.Time
	Limit      int
	Offset     int
}

type AggregateQuery struct {
	TenantID    string
	DeviceID    string
	MetricName  string
	Granularity telemetry.Granularity
	Start       time.Time
	End         time.Time
	Limit       int
	Offset      int
}

type WindowStatsQuery struct {
	TenantID   string
	DeviceID   string
	MetricName string
	Since      time.Time
	Until      time.Time
}

type WindowStats struct {
	Has   bool
	Count int
	Sum   float64
	Min   float64
	Max   float64
	Last  float64
}

func (s WindowStats) Avg() float64 {
	if s.Count == 0 {
		return 0
	}
	return s.Sum / float64(s.Count)
}
