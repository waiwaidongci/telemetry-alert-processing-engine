package routingconfig

type Validator interface{ Valid(string) bool }
type required struct{}

func (v *required) Valid(s string) bool { return s != "" }
func Disabled() Validator               { return nil }
