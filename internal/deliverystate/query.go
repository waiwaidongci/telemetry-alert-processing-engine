package deliverystate

func Active(s State) bool { return s == Queued }
