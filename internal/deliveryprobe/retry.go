package deliveryprobe

func Attempts(status int) int {
	if status >= 500 {
		return 3
	}
	return 1
}
