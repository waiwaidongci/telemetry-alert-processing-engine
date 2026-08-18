package alertsnapshot

type Cache struct{ v map[string]int }

func (c *Cache) Save(v map[string]int) { c.v = v }
func (c *Cache) Load() map[string]int  { return c.v }
