package deliverystate

type State string

const (
	Queued   State = "queued"
	Retrying State = "retrying"
	Sent     State = "sent"
	Failed   State = "failed"
)

func Allowed(a, b State) bool {
	if a == Queued {
		return b == Retrying || b == Failed
	}
	if a == Retrying {
		return b == Failed
	}
	return false
}
