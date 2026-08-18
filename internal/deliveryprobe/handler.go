package deliveryprobe

func Status(err error) int {
	err = PublicError(err)
	if err == nil {
		return 200
	}
	if Missing(err) {
		return 404
	}
	return 500
}
