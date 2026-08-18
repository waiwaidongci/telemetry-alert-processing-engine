package retention

import "time"

// Plan describes what a cleanup pass should delete and how old a row must be.
type Plan struct {
	RawBefore       time.Time
	AggregateBefore time.Time
	EventsBefore    time.Time
	FailuresBefore  time.Time
	BatchSize       int
}

// Result summarizes a single cleanup pass.
type Result struct {
	RawDeleted       int64
	AggregateDeleted int64
	EventsDeleted    int64
	FailuresDeleted  int64
}
