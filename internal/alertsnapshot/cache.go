package alertsnapshot

type Cache struct{ v map[string]int }

func (c *Cache) Save(v map[string]int) {
	out := make(map[string]int, len(v))
	for k, val := range v {
		out[k] = val
	}
	c.v = out
}

func (c *Cache) Load() map[string]int {
	out := make(map[string]int, len(c.v))
	for k, val := range c.v {
		out[k] = val
	}
	return out
}
