package alertsnapshot

func Total(v map[string]int) int {
	v = clone(v)
	n := 0
	for _, x := range v {
		n += x
	}
	return n
}
