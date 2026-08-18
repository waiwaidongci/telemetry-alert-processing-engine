package alertrule

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/alert"
	"github.com/example/telemetry-alert/internal/domain/device"
	"github.com/example/telemetry-alert/internal/system"
)

type Evaluator struct {
	rules    RuleRepository
	devices  DeviceReader
	states   StateRepository
	events   EventWriter
	metrics  MetricWindowReader
	dispatch Dispatcher
	clock    system.Clock
	ids      system.IDGenerator
	logger   *slog.Logger

	jobs    chan evaluationJob
	stop    chan struct{}
	wg      sync.WaitGroup
	workers int
}

type evaluationJob struct {
	tenantID string
	deviceID string
}

func NewEvaluator(
	rules RuleRepository,
	devices DeviceReader,
	states StateRepository,
	events EventWriter,
	metrics MetricWindowReader,
	dispatch Dispatcher,
	clock system.Clock,
	ids system.IDGenerator,
	logger *slog.Logger,
	workers int,
) *Evaluator {
	if workers <= 0 {
		workers = 2
	}
	return &Evaluator{
		rules:    rules,
		devices:  devices,
		states:   states,
		events:   events,
		metrics:  metrics,
		dispatch: dispatch,
		clock:    clock,
		ids:      ids,
		logger:   logger,
		jobs:     make(chan evaluationJob, 1024),
		stop:     make(chan struct{}),
		workers:  workers,
	}
}

func (e *Evaluator) Start() {
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker()
	}
}

func (e *Evaluator) Stop() {
	close(e.stop)
	e.wg.Wait()
}

func (e *Evaluator) worker() {
	defer e.wg.Done()
	for {
		select {
		case job := <-e.jobs:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			e.evaluateDevice(ctx, job.tenantID, job.deviceID)
			cancel()
		case <-e.stop:
			return
		}
	}
}

func (e *Evaluator) EvaluateDevice(_ context.Context, tenantID, deviceID string) {
	job := evaluationJob{tenantID: tenantID, deviceID: deviceID}
	select {
	case e.jobs <- job:
	default:
		// A full queue is better handled by the periodic sweep than by blocking
		// an ingest request. The sweep will catch up.
		e.logger.Warn("alert evaluation queue full; device will be covered by sweep",
			"tenant_id", tenantID, "device_id", deviceID)
	}
}

func (e *Evaluator) Sweep(ctx context.Context) error {
	rules, err := e.rules.ListEnabledAll(ctx)
	if err != nil {
		return fmt.Errorf("list enabled rules for sweep: %w", err)
	}
	for _, rule := range rules {
		if err := e.evaluateRule(ctx, rule); err != nil {
			e.logger.Error("evaluate rule", "rule_id", rule.ID, "error", err)
		}
	}
	return nil
}

func (e *Evaluator) evaluateDevice(ctx context.Context, tenantID, deviceID string) {
	rules, err := e.rules.ListEnabled(ctx, tenantID)
	if err != nil {
		e.logger.Error("list rules for device evaluation", "tenant_id", tenantID, "error", err)
		return
	}
	for _, rule := range rules {
		if rule.DeviceID != "" && rule.DeviceID != deviceID {
			continue
		}
		if rule.DeviceID == "" && rule.DeviceTag != "" {
			d, err := e.devices.Get(ctx, tenantID, deviceID)
			if err != nil || !deviceMatchesTag(d, rule.DeviceTag) {
				continue
			}
		}
		if err := e.evaluateRuleForDevice(ctx, rule, deviceID); err != nil {
			e.logger.Error("evaluate rule for device", "rule_id", rule.ID, "device_id", deviceID, "error", err)
		}
	}
}

func (e *Evaluator) evaluateRule(ctx context.Context, rule alert.Rule) error {
	var devices []device.Device
	if rule.DeviceID != "" {
		d, err := e.devices.Get(ctx, rule.TenantID, rule.DeviceID)
		if err != nil {
			return fmt.Errorf("get target device: %w", err)
		}
		devices = []device.Device{d}
	} else {
		const pageSize = 500
		for offset := 0; ; offset += pageSize {
			page, err := e.devices.List(ctx, rule.TenantID, "", pageSize, offset)
			if err != nil {
				return fmt.Errorf("list target devices: %w", err)
			}
			if rule.DeviceTag == "" {
				devices = append(devices, page...)
			} else {
				for _, d := range page {
					if deviceMatchesTag(d, rule.DeviceTag) {
						devices = append(devices, d)
					}
				}
			}
			if len(page) < pageSize {
				break
			}
		}
	}
	for _, d := range devices {
		if !d.IsActive() {
			continue
		}
		if err := e.evaluateRuleForDevice(ctx, rule, d.ID); err != nil {
			return fmt.Errorf("evaluate rule %s for device %s: %w", rule.ID, d.ID, err)
		}
	}
	return nil
}

func deviceMatchesTag(d device.Device, tag string) bool {
	if d.Tags != nil {
		for _, v := range d.Tags {
			if v == tag {
				return true
			}
		}
	}
	return d.Type == tag || d.Name == tag || d.Location == tag
}

func (e *Evaluator) evaluateRuleForDevice(ctx context.Context, rule alert.Rule, deviceID string) error {
	now := e.clock.Now().UTC()
	sample, err := e.metrics.ReadWindow(ctx, rule.TenantID, deviceID, rule.MetricName, now.Add(-rule.Window))
	if err != nil {
		return fmt.Errorf("read metric window: %w", err)
	}
	state, err := e.states.Get(ctx, rule.TenantID, rule.ID, deviceID, rule.MetricName)
	if err != nil && err != application.ErrNotFound {
		return fmt.Errorf("get alert state: %w", err)
	}
	if state.Since.IsZero() {
		state = alert.Instance{
			TenantID:   rule.TenantID,
			RuleID:     rule.ID,
			DeviceID:   deviceID,
			MetricName: rule.MetricName,
			State:      alert.StateNormal,
		}
	}
	state.LastValue = sample.Value
	state.UpdatedAt = now

	conditionValue := sample.Avg
	if !sample.Has {
		if state.State != alert.StateNormal {
			return e.recover(ctx, rule, deviceID, state, now)
		}
		return nil
	}

	if compare(conditionValue, rule.Operator, rule.Threshold) {
		switch state.State {
		case alert.StateNormal, alert.StateRecovered:
			state.State = alert.StatePending
			state.Since = now
			if err := e.states.Upsert(ctx, state); err != nil {
				return fmt.Errorf("save pending state: %w", err)
			}
		case alert.StatePending:
			if !now.Before(state.Since.Add(rule.Duration)) && now.After(state.LastEventAt.Add(rule.Cooldown)) {
				if err := e.emit(ctx, rule, deviceID, state, alert.EventTriggered); err != nil {
					return err
				}
				state.State = alert.StateTriggered
				state.LastEventAt = now
				if err := e.states.Upsert(ctx, state); err != nil {
					return fmt.Errorf("save triggered state: %w", err)
				}
			}
		case alert.StateTriggered:
			if now.After(state.LastEventAt.Add(rule.Cooldown)) {
				if err := e.emit(ctx, rule, deviceID, state, alert.EventContinuing); err != nil {
					return err
				}
				state.LastEventAt = now
				if err := e.states.Upsert(ctx, state); err != nil {
					return fmt.Errorf("save continuing state: %w", err)
				}
			}
		}
		return nil
	}

	if state.State != alert.StateNormal {
		return e.recover(ctx, rule, deviceID, state, now)
	}
	return nil
}

func (e *Evaluator) recover(ctx context.Context, rule alert.Rule, deviceID string, state alert.Instance, now time.Time) error {
	if err := e.emit(ctx, rule, deviceID, state, alert.EventRecovered); err != nil {
		return err
	}
	state.State = alert.StateRecovered
	state.LastEventAt = now
	if err := e.states.Upsert(ctx, state); err != nil {
		return fmt.Errorf("save recovered state: %w", err)
	}
	return nil
}

func (e *Evaluator) emit(ctx context.Context, rule alert.Rule, deviceID string, state alert.Instance, eventType alert.EventType) error {
	event := alert.Event{
		ID:         e.ids.New(),
		TenantID:   rule.TenantID,
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		DeviceID:   deviceID,
		MetricName: rule.MetricName,
		Value:      state.LastValue,
		EventType:  eventType,
		Level:      rule.Level,
		Channels:   rule.Channels,
		Message:    eventMessage(rule, deviceID, state.LastValue, eventType),
		CreatedAt:  e.clock.Now().UTC(),
	}
	if err := e.events.Create(ctx, event); err != nil {
		return fmt.Errorf("create alert event: %w", err)
	}
	// Dispatch is intentionally asynchronous at the notification layer; this
	// call should only enqueue work and never block the evaluation loop.
	e.dispatch.Dispatch(ctx, event)
	return nil
}

func compare(value float64, op alert.Operator, threshold float64) bool {
	switch op {
	case alert.OpGreaterThan:
		return value > threshold
	case alert.OpGreaterThanOrEqual:
		return value >= threshold
	case alert.OpLessThan:
		return value < threshold
	case alert.OpLessThanOrEqual:
		return value <= threshold
	case alert.OpEqual:
		return value == threshold
	default:
		return false
	}
}

func eventMessage(rule alert.Rule, deviceID string, value float64, eventType alert.EventType) string {
	switch eventType {
	case alert.EventTriggered:
		return fmt.Sprintf("rule %q triggered for device %s: %s %.3f", rule.Name, deviceID, rule.MetricName, value)
	case alert.EventContinuing:
		return fmt.Sprintf("rule %q continues for device %s: %s %.3f", rule.Name, deviceID, rule.MetricName, value)
	case alert.EventRecovered:
		return fmt.Sprintf("rule %q recovered for device %s: %s %.3f", rule.Name, deviceID, rule.MetricName, value)
	default:
		return fmt.Sprintf("rule %q event for device %s", rule.Name, deviceID)
	}
}
