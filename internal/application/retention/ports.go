package retention

import (
	"context"

	"github.com/example/telemetry-alert/internal/domain/retention"
)

type Cleaner interface {
	Clean(ctx context.Context, plan retention.Plan) (retention.Result, error)
}
