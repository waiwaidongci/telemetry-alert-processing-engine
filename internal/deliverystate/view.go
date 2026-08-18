package deliverystate

func Visible(s State) bool { return s != Retrying }
