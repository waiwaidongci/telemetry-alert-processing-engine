package telemetry

import "time"

// Point is a single telemetry sample accepted from a device.
type Point struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	DeviceID       string            `json:"device_id"`
	MetricName     string            `json:"metric_name"`
	Value          float64           `json:"value"`
	Timestamp      time.Time         `json:"timestamp"`
	Labels         map[string]string `json:"labels,omitempty"`
	ReceivedAt     time.Time         `json:"received_at"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

// AggregateBucket is a compact representation of a fixed time bucket.
type AggregateBucket struct {
	TenantID    string    `json:"tenant_id"`
	DeviceID    string    `json:"device_id"`
	MetricName  string    `json:"metric_name"`
	Granularity string    `json:"granularity"`
	BucketTime  time.Time `json:"bucket_time"`
	Count       int64     `json:"count"`
	Sum         float64   `json:"sum"`
	Min         float64   `json:"min"`
	Max         float64   `json:"max"`
}

func (b AggregateBucket) Avg() float64 {
	if b.Count == 0 {
		return 0
	}
	return b.Sum / float64(b.Count)
}

type Granularity string

const (
	GranularityRaw     Granularity = "raw"
	Granularity1Minute Granularity = "1m"
	Granularity5Minute Granularity = "5m"
	Granularity1Hour   Granularity = "1h"
	Granularity1Day    Granularity = "1d"
)
