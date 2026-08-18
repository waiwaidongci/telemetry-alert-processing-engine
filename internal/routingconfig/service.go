package routingconfig

func Register(c *Config, key, value string) { c.Routes[key] = value }
func Accept(v Validator, route string) bool {
	if v != nil {
		return v.Valid(route)
	}
	return route != ""
}
