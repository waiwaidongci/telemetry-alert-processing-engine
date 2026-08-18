package telemetry

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/telemetry"
	"github.com/example/telemetry-alert/internal/system"
)

type Config struct {
	MaxBatchSize      int
	MaxPointAge       time.Duration
	MaxFutureSkew     time.Duration
	QueryDefaultLimit int
	QueryMaxLimit     int
}

type Service struct {
	ingest IngestRepository
	query  QueryRepository
	eval   Evaluator
	limits RateLimiter
	ids    system.IDGenerator
	clock  system.Clock
	cfg    Config
}

func NewService(ingest IngestRepository, query QueryRepository, eval Evaluator, limits RateLimiter, ids system.IDGenerator, clock system.Clock, cfg Config) *Service {
	return &Service{ingest: ingest, query: query, eval: eval, limits: limits, ids: ids, clock: clock, cfg: cfg}
}

func (s *Service) IngestSingle(ctx context.Context, tenantID, deviceID string, in IngestInput) (telemetry.Point, error) {
	if !s.limits.Allow(ctx, tenantID+"/"+deviceID, 1) {
		return telemetry.Point{}, application.ErrRateLimited
	}
	point, err := s.ingestOne(ctx, tenantID, deviceID, in)
	if err != nil {
		return telemetry.Point{}, err
	}
	if _, err := s.ingest.InsertBatch(ctx, []telemetry.Point{point}); err != nil {
		return telemetry.Point{}, fmt.Errorf("insert telemetry: %w", err)
	}
	s.eval.EvaluateDevice(ctx, tenantID, deviceID)
	return point, nil
}

func (s *Service) IngestBatch(ctx context.Context, tenantID, deviceID string, inputs []IngestInput) ([]telemetry.Point, error) {
	if len(inputs) == 0 {
		return nil, application.ValidationError{Field: "batch", Message: "must not be empty"}
	}
	if len(inputs) > s.cfg.MaxBatchSize {
		return nil, application.ValidationError{Field: "batch", Message: fmt.Sprintf("exceeds max batch size %d", s.cfg.MaxBatchSize)}
	}
	if !s.limits.Allow(ctx, tenantID+"/"+deviceID, len(inputs)) {
		return nil, application.ErrRateLimited
	}
	points := make([]telemetry.Point, 0, len(inputs))
	for _, in := range inputs {
		point, err := s.ingestOne(ctx, tenantID, deviceID, in)
		if err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	inserted, err := s.ingest.InsertBatch(ctx, points)
	if err != nil {
		return nil, fmt.Errorf("insert telemetry batch: %w", err)
	}
	if inserted == 0 {
		return nil, application.ErrConflict
	}
	s.eval.EvaluateDevice(ctx, tenantID, deviceID)
	return points, nil
}

func (s *Service) ingestOne(ctx context.Context, tenantID, deviceID string, in IngestInput) (telemetry.Point, error) {
	if tenantID == "" || deviceID == "" {
		return telemetry.Point{}, application.ValidationError{Field: "device_id", Message: "tenant and device identifiers are required"}
	}
	if strings.TrimSpace(in.MetricName) == "" {
		return telemetry.Point{}, application.ValidationError{Field: "metric_name", Message: "must not be empty"}
	}
	if math.IsNaN(in.Value) || math.IsInf(in.Value, 0) {
		return telemetry.Point{}, application.ValidationError{Field: "value", Message: "must be a finite number"}
	}
	if in.Timestamp.IsZero() {
		return telemetry.Point{}, application.ValidationError{Field: "timestamp", Message: "must not be empty"}
	}
	now := s.clock.Now().UTC()
	if in.Timestamp.After(now.Add(s.cfg.MaxFutureSkew)) {
		return telemetry.Point{}, application.ValidationError{Field: "timestamp", Message: "too far in the future"}
	}
	if in.Timestamp.Before(now.Add(-s.cfg.MaxPointAge)) {
		return telemetry.Point{}, application.ValidationError{Field: "timestamp", Message: "too far in the past"}
	}
	if in.IdempotencyKey != "" {
		exists, err := s.ingest.KeyExists(ctx, tenantID, in.IdempotencyKey)
		if err != nil {
			return telemetry.Point{}, fmt.Errorf("check idempotency key: %w", err)
		}
		if exists {
			return telemetry.Point{}, application.ErrConflict
		}
	}
	labels := cloneLabels(in.Labels)
	return telemetry.Point{
		ID:             s.ids.New(),
		TenantID:       tenantID,
		DeviceID:       deviceID,
		MetricName:     in.MetricName,
		Value:          in.Value,
		Timestamp:      in.Timestamp.UTC(),
		Labels:         labels,
		ReceivedAt:     now,
		IdempotencyKey: in.IdempotencyKey,
	}, nil
}

func (s *Service) QueryRaw(ctx context.Context, q RawQuery) ([]telemetry.Point, error) {
	if q.TenantID == "" {
		return nil, application.ValidationError{Field: "tenant_id", Message: "must not be empty"}
	}
	if q.Start.IsZero() || q.End.IsZero() || !q.End.After(q.Start) {
		return nil, application.ValidationError{Field: "time_range", Message: "start must be before end"}
	}
	q.Limit = boundedLimit(q.Limit, s.cfg.QueryDefaultLimit, s.cfg.QueryMaxLimit)
	if q.Offset < 0 {
		q.Offset = 0
	}
	points, err := s.query.ListRaw(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query raw telemetry: %w", err)
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Timestamp.Before(points[j].Timestamp) })
	return points, nil
}

func (s *Service) QueryAggregates(ctx context.Context, q AggregateQuery) ([]telemetry.AggregateBucket, error) {
	if q.TenantID == "" {
		return nil, application.ValidationError{Field: "tenant_id", Message: "must not be empty"}
	}
	if !validGranularity(q.Granularity) {
		return nil, application.ValidationError{Field: "granularity", Message: "must be 1m, 5m, 1h, or 1d"}
	}
	if q.Start.IsZero() || q.End.IsZero() || !q.End.After(q.Start) {
		return nil, application.ValidationError{Field: "time_range", Message: "start must be before end"}
	}
	q.Limit = boundedLimit(q.Limit, s.cfg.QueryDefaultLimit, s.cfg.QueryMaxLimit)
	if q.Offset < 0 {
		q.Offset = 0
	}
	buckets, err := s.query.ListAggregates(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query aggregate telemetry: %w", err)
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].BucketTime.Before(buckets[j].BucketTime) })
	return buckets, nil
}

func boundedLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

func validGranularity(g telemetry.Granularity) bool {
	switch g {
	case telemetry.Granularity1Minute, telemetry.Granularity5Minute, telemetry.Granularity1Hour, telemetry.Granularity1Day:
		return true
	default:
		return false
	}
}

func cloneLabels(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
