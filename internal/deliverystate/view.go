package deliverystate

func Visible(s State) bool { return s == Queued || s == Retrying || s == Sent || s == Failed }
