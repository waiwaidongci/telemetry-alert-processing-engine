package ratelimit

import (
	"context"
	"sync"
	"time"
)

// TokenBucketLimiter is an in-memory rate limiter scoped by an arbitrary key.
// It is deliberately simple and suitable for a single process deployment.
type TokenBucketLimiter struct {
	rate     float64
	capacity float64
	mu       sync.Mutex
	buckets  map[string]*bucket
	now      func() time.Time
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

func NewTokenBucketLimiter(rate, capacity int) *TokenBucketLimiter {
	if rate <= 0 {
		rate = 1
	}
	if capacity <= 0 {
		capacity = rate
	}
	return &TokenBucketLimiter{
		rate:     float64(rate),
		capacity: float64(capacity),
		buckets:  map[string]*bucket{},
		now:      time.Now,
	}
}

func (l *TokenBucketLimiter) Allow(_ context.Context, key string, n int) bool {
	if n <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: l.capacity, lastSeen: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastSeen).Seconds()
		b.tokens += elapsed * l.rate
		if b.tokens > l.capacity {
			b.tokens = l.capacity
		}
		b.lastSeen = now
	}
	if b.tokens < float64(n) {
		return false
	}
	b.tokens -= float64(n)
	return true
}
