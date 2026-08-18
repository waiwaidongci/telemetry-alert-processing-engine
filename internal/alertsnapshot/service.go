package alertsnapshot

func Current(s *Store) map[string]int { return s.Snapshot() }
