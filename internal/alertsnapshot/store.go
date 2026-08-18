package alertsnapshot

import "sync"

type Store struct {
	mu     sync.RWMutex
	values map[string]int
}

func New() *Store { return &Store{values: map[string]int{"open": 1}} }
func clone(v map[string]int) map[string]int {
	o := make(map[string]int, len(v))
	for k, n := range v {
		o[k] = n
	}
	return o
}
func (s *Store) Snapshot() map[string]int { s.mu.RLock(); defer s.mu.RUnlock(); return clone(s.values) }
func (s *Store) Set(k string, v int)      { s.mu.Lock(); defer s.mu.Unlock(); s.values[k] = v }
