package routingconfig

type Validator interface{ Valid(string) bool }
type required struct{}

func (v *required) Valid(s string) bool {
	if v == nil {
		return true
	}
	return s != ""
}
func Disabled() Validator { var v *required; return v }
