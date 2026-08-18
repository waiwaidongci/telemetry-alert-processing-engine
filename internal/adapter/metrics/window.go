package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/example/telemetry-alert/internal/application/alertrule"
	"github.com/example/telemetry-alert/internal/application/telemetry"
)

// WindowReader adapts telemetry query storage into the narrow metric reader
// contract used by the alert evaluator.
type WindowReader struct {
	repo telemetry.QueryRepository
}

func NewWindowReader(repo telemetry.QueryRepository) *WindowReader {
	return &WindowReader{repo: repo}
}

func (r *WindowReader) ReadWindow(ctx context.Context, tenantID, deviceID, metricName string, since time.Time) (alertrule.MetricSample, error) {
	until := time.Now().UTC()
	stats, err := r.repo.WindowStats(ctx, telemetry.WindowStatsQuery{
		TenantID:   tenantID,
		DeviceID:   deviceID,
		MetricName: metricName,
		Since:      since,
		Until:      until,
	})
	if err != nil {
		return alertrule.MetricSample{}, fmt.Errorf("read telemetry window stats: %w", err)
	}
	return alertrule.MetricSample{
		Has:   stats.Has,
		Value: stats.Last,
		Avg:   stats.Avg(),
		Count: stats.Count,
	}, nil
}
