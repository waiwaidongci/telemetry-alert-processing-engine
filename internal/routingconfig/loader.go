package routingconfig

type Config struct{ Routes map[string]string }

func Load() Config { return Config{} }
