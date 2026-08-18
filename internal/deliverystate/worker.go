package deliverystate

func FinishRetry(ok bool) State {
	if ok {
		return Retrying
	}
	return Failed
}
