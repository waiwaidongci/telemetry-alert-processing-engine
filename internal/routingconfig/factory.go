package routingconfig

func DefaultValidator() Validator { return &required{} }
