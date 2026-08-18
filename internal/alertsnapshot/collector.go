package alertsnapshot

func Total(v map[string]int) int {
	n := 0
	for _, x := range v {
		n += x
	}
	return n
}
