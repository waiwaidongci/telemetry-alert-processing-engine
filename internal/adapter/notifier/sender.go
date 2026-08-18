package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/example/telemetry-alert/internal/domain/alert"
	"github.com/example/telemetry-alert/internal/domain/notification"
)

type Sender struct {
	client     *http.Client
	webhookURL string
	logger     *slog.Logger
}

func NewSender(webhookURL string, timeout time.Duration, logger *slog.Logger) *Sender {
	return &Sender{
		client:     &http.Client{Timeout: timeout},
		webhookURL: webhookURL,
		logger:     logger,
	}
}

func (s *Sender) Send(ctx context.Context, event alert.Event, channel notification.Channel) error {
	switch channel {
	case notification.ChannelLog:
		s.logger.Info("alert notification",
			"event_id", event.ID,
			"tenant_id", event.TenantID,
			"rule_id", event.RuleID,
			"device_id", event.DeviceID,
			"metric_name", event.MetricName,
			"value", event.Value,
			"event_type", event.EventType,
			"level", event.Level,
		)
		return nil
	case notification.ChannelWebhook:
		return s.sendWebhook(ctx, event)
	default:
		return fmt.Errorf("unsupported notification channel %q", channel)
	}
}

func (s *Sender) sendWebhook(ctx context.Context, event alert.Event) error {
	if s.webhookURL == "" {
		return fmt.Errorf("webhook url is not configured")
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook returned %s: %s", resp.Status, string(msg))
	}
	return nil
}
