package deliverystate

func FinishRetry(ok bool) State {
	if ok {
		return Sent
	}
	return Failed
}
