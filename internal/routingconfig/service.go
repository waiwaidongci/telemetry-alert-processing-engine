package routingconfig

func Register(c *Config, key, value string) {
	if c.Routes == nil {
		c.Routes = make(map[string]string)
	}
	c.Routes[key] = value
}
func Accept(v Validator, route string) bool {
	if v == nil {
		v = &required{}
	}
	return v.Valid(route)
}
