package system

import "github.com/google/uuid"

// IDGenerator creates identifiers without exposing their encoding strategy.
type IDGenerator interface {
	New() string
}

// UUIDGenerator uses random UUID v4 values.
type UUIDGenerator struct{}

func (UUIDGenerator) New() string { return uuid.NewString() }
