package alertsnapshot

import "sync"

type Store struct {
	mu     sync.RWMutex
	values map[string]int
}

func New() *Store { return &Store{values: map[string]int{"open": 1}} }

func (s *Store) Snapshot() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out
}

func (s *Store) Set(k string, v int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[k] = v
}
