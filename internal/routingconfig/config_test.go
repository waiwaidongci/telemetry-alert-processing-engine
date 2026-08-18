package routingconfig

import "testing"

func TestDefaultsAreUsable(t *testing.T) {
	c := Load()
	defer func() {
		if recover() != nil {
			t.Fatal("default routes panic")
		}
	}()
	Register(&c, "critical", "pager")
	if Accept(Disabled(), "") {
		t.Fatal("empty route accepted")
	}
}
