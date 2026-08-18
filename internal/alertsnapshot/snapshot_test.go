package alertsnapshot

import "testing"

func TestSnapshotsRemainStable(t *testing.T) {
	s := New()
	v := Current(s)
	c := &Cache{}
	c.Save(v)
	s.Set("closed", 2)
	if len(v) != 1 {
		t.Fatalf("snapshot changed")
	}
	c.Load()["open"] = 9
	if v["open"] != 1 {
		t.Fatalf("cache leaked mutation")
	}
}
