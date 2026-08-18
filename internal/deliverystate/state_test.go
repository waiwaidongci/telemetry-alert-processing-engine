package deliverystate

import "testing"

func TestSuccessfulRetryReachesSent(t *testing.T) {
	if !Allowed(Retrying, Sent) {
		t.Fatal("transition denied")
	}
	if FinishRetry(true) != Sent {
		t.Fatal("wrong final state")
	}
	if !Active(Retrying) {
		t.Fatal("retry missing from active view")
	}
}
