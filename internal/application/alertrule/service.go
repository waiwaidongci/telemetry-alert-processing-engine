package alertrule

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/domain/alert"
	"github.com/example/telemetry-alert/internal/system"
)

type CreateInput struct {
	Name       string
	DeviceID   string
	DeviceTag  string
	MetricName string
	Operator   alert.Operator
	Threshold  float64
	Window     time.Duration
	Duration   time.Duration
	Level      alert.Level
	Cooldown   time.Duration
	Channels   []string
}

type Service struct {
	rules  RuleRepository
	states StateRepository
	ids    system.IDGenerator
	clock  system.Clock
}

func NewService(rules RuleRepository, states StateRepository, ids system.IDGenerator, clock system.Clock) *Service {
	return &Service{rules: rules, states: states, ids: ids, clock: clock}
}

func (s *Service) Create(ctx context.Context, tenantID string, in CreateInput) (alert.Rule, error) {
	if err := validateCreate(tenantID, in); err != nil {
		return alert.Rule{}, err
	}
	now := s.clock.Now().UTC()
	rule := alert.Rule{
		ID:         s.ids.New(),
		TenantID:   tenantID,
		Name:       in.Name,
		DeviceID:   in.DeviceID,
		DeviceTag:  in.DeviceTag,
		MetricName: in.MetricName,
		Operator:   in.Operator,
		Threshold:  in.Threshold,
		Window:     in.Window,
		Duration:   in.Duration,
		Level:      in.Level,
		Cooldown:   in.Cooldown,
		Channels:   cleanChannels(in.Channels),
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.rules.Create(ctx, rule); err != nil {
		return alert.Rule{}, fmt.Errorf("create alert rule: %w", err)
	}
	return rule, nil
}

func (s *Service) Get(ctx context.Context, tenantID, id string) (alert.Rule, error) {
	if tenantID == "" || id == "" {
		return alert.Rule{}, application.ValidationError{Field: "id", Message: "tenant and rule id are required"}
	}
	rule, err := s.rules.Get(ctx, tenantID, id)
	if err != nil {
		return alert.Rule{}, fmt.Errorf("get alert rule: %w", err)
	}
	return rule, nil
}

func (s *Service) List(ctx context.Context, tenantID string, limit, offset int) ([]alert.Rule, error) {
	if tenantID == "" {
		return nil, application.ValidationError{Field: "tenant_id", Message: "must not be empty"}
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	list, err := s.rules.List(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list alert rules: %w", err)
	}
	return list, nil
}

func (s *Service) Update(ctx context.Context, tenantID, id string, in CreateInput) (alert.Rule, error) {
	if err := validateCreate(tenantID, in); err != nil {
		return alert.Rule{}, err
	}
	rule, err := s.rules.Get(ctx, tenantID, id)
	if err != nil {
		return alert.Rule{}, fmt.Errorf("get alert rule for update: %w", err)
	}
	rule.Name = in.Name
	rule.DeviceID = in.DeviceID
	rule.DeviceTag = in.DeviceTag
	rule.MetricName = in.MetricName
	rule.Operator = in.Operator
	rule.Threshold = in.Threshold
	rule.Window = in.Window
	rule.Duration = in.Duration
	rule.Level = in.Level
	rule.Cooldown = in.Cooldown
	rule.Channels = cleanChannels(in.Channels)
	rule.UpdatedAt = s.clock.Now().UTC()
	if err := s.rules.Update(ctx, rule); err != nil {
		return alert.Rule{}, fmt.Errorf("update alert rule: %w", err)
	}
	if err := s.states.DeleteByRule(ctx, tenantID, id); err != nil {
		return alert.Rule{}, fmt.Errorf("reset alert states: %w", err)
	}
	return rule, nil
}

func (s *Service) SetEnabled(ctx context.Context, tenantID, id string, enabled bool) (alert.Rule, error) {
	rule, err := s.rules.Get(ctx, tenantID, id)
	if err != nil {
		return alert.Rule{}, fmt.Errorf("get alert rule for enable: %w", err)
	}
	rule.Enabled = enabled
	rule.UpdatedAt = s.clock.Now().UTC()
	if err := s.rules.Update(ctx, rule); err != nil {
		return alert.Rule{}, fmt.Errorf("update alert rule enabled: %w", err)
	}
	if !enabled {
		if err := s.states.DeleteByRule(ctx, tenantID, id); err != nil {
			return alert.Rule{}, fmt.Errorf("reset disabled alert states: %w", err)
		}
	}
	return rule, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, id string) error {
	if tenantID == "" || id == "" {
		return application.ValidationError{Field: "id", Message: "tenant and rule id are required"}
	}
	if err := s.rules.Delete(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete alert rule: %w", err)
	}
	if err := s.states.DeleteByRule(ctx, tenantID, id); err != nil {
		return fmt.Errorf("delete alert states: %w", err)
	}
	return nil
}

func validateCreate(tenantID string, in CreateInput) error {
	if tenantID == "" {
		return application.ValidationError{Field: "tenant_id", Message: "must not be empty"}
	}
	if strings.TrimSpace(in.Name) == "" {
		return application.ValidationError{Field: "name", Message: "must not be empty"}
	}
	if strings.TrimSpace(in.MetricName) == "" {
		return application.ValidationError{Field: "metric_name", Message: "must not be empty"}
	}
	if !validOperator(in.Operator) {
		return application.ValidationError{Field: "operator", Message: "must be gt, gte, lt, lte, or eq"}
	}
	if !validLevel(in.Level) {
		return application.ValidationError{Field: "level", Message: "must be info, warning, or critical"}
	}
	if in.Window <= 0 {
		return application.ValidationError{Field: "window", Message: "must be positive"}
	}
	if in.Duration < 0 {
		return application.ValidationError{Field: "duration", Message: "must not be negative"}
	}
	if in.Cooldown < 0 {
		return application.ValidationError{Field: "cooldown", Message: "must not be negative"}
	}
	if len(in.Channels) == 0 {
		return application.ValidationError{Field: "channels", Message: "must not be empty"}
	}
	return nil
}

func validOperator(op alert.Operator) bool {
	switch op {
	case alert.OpGreaterThan, alert.OpGreaterThanOrEqual, alert.OpLessThan, alert.OpLessThanOrEqual, alert.OpEqual:
		return true
	default:
		return false
	}
}

func validLevel(l alert.Level) bool {
	switch l {
	case alert.LevelInfo, alert.LevelWarning, alert.LevelCritical:
		return true
	default:
		return false
	}
}

func cleanChannels(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, c := range in {
		c = strings.ToLower(strings.TrimSpace(c))
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	if len(out) == 0 {
		out = append(out, "log")
	}
	return out
}
