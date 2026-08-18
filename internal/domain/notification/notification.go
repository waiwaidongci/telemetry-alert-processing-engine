package notification

import "time"

type Channel string

const (
	ChannelWebhook Channel = "webhook"
	ChannelLog     Channel = "log"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

// Attempt tracks delivery of one alert event through one channel.
type Attempt struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	EventID       string    `json:"event_id"`
	Channel       Channel   `json:"channel"`
	Status        Status    `json:"status"`
	Attempts      int       `json:"attempts"`
	NextAttemptAt time.Time `json:"next_attempt_at,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
