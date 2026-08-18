package api

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	requests     atomic.Int64
	errors       atomic.Int64
	ingestPoints atomic.Int64
}

func NewMetrics() *Metrics { return &Metrics{} }

func (m *Metrics) Request()        { m.requests.Add(1) }
func (m *Metrics) Error()          { m.errors.Add(1) }
func (m *Metrics) AddIngest(n int) { m.ingestPoints.Add(int64(n)) }

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# HELP telemetry_http_requests_total Total HTTP requests.\n")
		fmt.Fprintf(w, "# TYPE telemetry_http_requests_total counter\ntelemetry_http_requests_total %d\n", m.requests.Load())
		fmt.Fprintf(w, "# HELP telemetry_http_errors_total Total HTTP error responses.\n")
		fmt.Fprintf(w, "# TYPE telemetry_http_errors_total counter\ntelemetry_http_errors_total %d\n", m.errors.Load())
		fmt.Fprintf(w, "# HELP telemetry_ingest_points_total Total telemetry points accepted.\n")
		fmt.Fprintf(w, "# TYPE telemetry_ingest_points_total counter\ntelemetry_ingest_points_total %d\n", m.ingestPoints.Load())
	})
}
