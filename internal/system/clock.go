package system

import "time"

// Clock abstracts time so services can be tested deterministically.
type Clock interface {
	Now() time.Time
}

// RealClock is the production clock backed by time.Now.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
