package deliveryprobe

func Status(err error) int {
	if err == nil {
		return 200
	}
	if Missing(err) {
		return 404
	}
	return 500
}
