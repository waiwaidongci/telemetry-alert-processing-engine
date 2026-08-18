package clock

import (
	"github.com/example/telemetry-alert/internal/system"
)

// Real returns the production clock implementation.
func Real() system.Clock {
	return system.RealClock{}
}
