package alertsnapshot

func Current(s *Store) map[string]int { return clone(s.Snapshot()) }
