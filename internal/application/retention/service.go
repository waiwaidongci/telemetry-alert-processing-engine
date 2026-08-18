package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/example/telemetry-alert/internal/domain/retention"
	"github.com/example/telemetry-alert/internal/system"
)

type Config struct {
	RawRetention       time.Duration
	AggregateRetention time.Duration
	EventRetention     time.Duration
	FailureRetention   time.Duration
	DeleteBatchSize    int
}

type Service struct {
	cleaner Cleaner
	clock   system.Clock
	cfg     Config
}

func NewService(cleaner Cleaner, clock system.Clock, cfg Config) *Service {
	return &Service{cleaner: cleaner, clock: clock, cfg: cfg}
}

func (s *Service) Run(ctx context.Context) (retention.Result, error) {
	now := s.clock.Now().UTC()
	plan := retention.Plan{
		RawBefore:       now.Add(-s.cfg.RawRetention),
		AggregateBefore: now.Add(-s.cfg.AggregateRetention),
		EventsBefore:    now.Add(-s.cfg.EventRetention),
		FailuresBefore:  now.Add(-s.cfg.FailureRetention),
		BatchSize:       s.cfg.DeleteBatchSize,
	}
	result, err := s.cleaner.Clean(ctx, plan)
	if err != nil {
		return retention.Result{}, fmt.Errorf("run retention cleanup: %w", err)
	}
	return result, nil
}
