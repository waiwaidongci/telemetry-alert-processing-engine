package deliveryprobe

func Attempts(status int) int {
	switch {
	case status >= 500:
		return 3
	case status >= 400:
		return 1
	default:
		return 1
	}
}
