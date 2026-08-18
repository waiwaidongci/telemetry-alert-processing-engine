package notification

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/alert"
	"github.com/example/telemetry-alert/internal/domain/notification"
	"github.com/example/telemetry-alert/internal/system"
)

type Dispatcher struct {
	repo   Repository
	sender Sender
	events EventLoader
	ids    system.IDGenerator
	clock  system.Clock
	logger *slog.Logger
	cfg    Config

	jobs chan string
	stop chan struct{}
	wg   sync.WaitGroup
}

func NewDispatcher(repo Repository, sender Sender, events EventLoader, ids system.IDGenerator, clock system.Clock, logger *slog.Logger, cfg Config) *Dispatcher {
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 2
	}
	return &Dispatcher{
		repo:   repo,
		sender: sender,
		events: events,
		ids:    ids,
		clock:  clock,
		logger: logger,
		cfg:    cfg,
		jobs:   make(chan string, 4096),
		stop:   make(chan struct{}),
	}
}

func (d *Dispatcher) Start() {
	for i := 0; i < d.cfg.WorkerCount; i++ {
		d.wg.Add(1)
		go d.worker()
	}
}

func (d *Dispatcher) Stop() {
	close(d.stop)
	d.wg.Wait()
}

// Dispatch creates pending attempts and enqueues their delivery. Database work
// is intentionally small and no external network call happens here, so alert
// evaluation remains non-blocking.
func (d *Dispatcher) Dispatch(_ context.Context, event alert.Event) {
	for _, channelName := range uniqueChannels(event.Channels) {
		channel := notification.Channel(channelName)
		if channel != notification.ChannelWebhook && channel != notification.ChannelLog {
			continue
		}
		now := d.clock.Now().UTC()
		attempt := notification.Attempt{
			ID:            d.ids.New(),
			TenantID:      event.TenantID,
			EventID:       event.ID,
			Channel:       channel,
			Status:        notification.StatusPending,
			Attempts:      0,
			NextAttemptAt: now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := d.repo.Create(context.Background(), attempt); err != nil {
			d.logger.Error("create notification attempt", "event_id", event.ID, "channel", channel, "error", err)
			continue
		}
		select {
		case d.jobs <- attempt.ID:
		default:
			// Pending row will be picked up by RetryPending.
			d.logger.Warn("notification queue full; pending row left for retry", "attempt_id", attempt.ID)
		}
	}
}

func (d *Dispatcher) RetryPending(ctx context.Context) error {
	before := d.clock.Now().UTC()
	attempts, err := d.repo.ListPending(ctx, before, 500)
	if err != nil {
		return fmt.Errorf("list pending notification attempts: %w", err)
	}
	for _, attempt := range attempts {
		select {
		case d.jobs <- attempt.ID:
		default:
			return nil
		}
	}
	return nil
}

func (d *Dispatcher) ListFailures(ctx context.Context, tenantID string, limit, offset int) ([]notification.Attempt, error) {
	if tenantID == "" {
		return nil, application.ValidationError{Field: "tenant_id", Message: "must not be empty"}
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	list, err := d.repo.ListFailures(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list notification failures: %w", err)
	}
	return list, nil
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for {
		select {
		case id := <-d.jobs:
			d.process(id)
		case <-d.stop:
			return
		}
	}
}

func (d *Dispatcher) process(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.RequestTimeout*2)
	defer cancel()
	attempt, err := d.repo.Get(ctx, id)
	if err != nil {
		d.logger.Error("get notification attempt", "attempt_id", id, "error", err)
		return
	}
	if attempt.Status == notification.StatusSucceeded || attempt.Attempts >= d.cfg.MaxAttempts {
		if attempt.Status == notification.StatusPending {
			attempt.Status = notification.StatusFailed
			attempt.UpdatedAt = d.clock.Now().UTC()
			_ = d.repo.Update(ctx, attempt)
		}
		return
	}
	event, err := d.eventByID(ctx, attempt.EventID)
	if err != nil {
		d.logger.Error("load event for notification", "attempt_id", id, "event_id", attempt.EventID, "error", err)
		return
	}
	attempt.Attempts++
	if err := d.sender.Send(ctx, event, attempt.Channel); err != nil {
		attempt.LastError = err.Error()
		attempt.Status = notification.StatusFailed
		if attempt.Attempts < d.cfg.MaxAttempts {
			attempt.Status = notification.StatusPending
			attempt.NextAttemptAt = d.clock.Now().UTC().Add(backoff(d.cfg.InitialBackoff, d.cfg.MaxBackoff, attempt.Attempts))
		}
		attempt.UpdatedAt = d.clock.Now().UTC()
		if err := d.repo.Update(ctx, attempt); err != nil {
			d.logger.Error("update failed notification attempt", "attempt_id", id, "error", err)
		}
		return
	}
	attempt.Status = notification.StatusSucceeded
	attempt.LastError = ""
	attempt.UpdatedAt = d.clock.Now().UTC()
	if err := d.repo.Update(ctx, attempt); err != nil {
		d.logger.Error("update successful notification attempt", "attempt_id", id, "error", err)
	}
}

func (d *Dispatcher) eventByID(ctx context.Context, eventID string) (alert.Event, error) {
	event, err := d.events.Get(ctx, eventID)
	if err != nil {
		return alert.Event{}, fmt.Errorf("load alert event: %w", err)
	}
	return event, nil
}

func uniqueChannels(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, c := range in {
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

func backoff(initial, max time.Duration, attempt int) time.Duration {
	if initial <= 0 {
		initial = time.Second
	}
	if max < initial {
		max = initial
	}
	delay := initial
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= max {
			return max
		}
	}
	return delay
}
