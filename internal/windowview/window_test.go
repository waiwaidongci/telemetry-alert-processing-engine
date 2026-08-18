package windowview

import (
	"reflect"
	"testing"
)

func TestWindowDoesNotRewriteReadings(t *testing.T) {
	src := []Point{{1}, {-1}, {2}}
	want := append([]Point(nil), src...)
	w := New(Positive(src))
	_ = Merge(w, []Point{{3}})
	if !reflect.DeepEqual(src, want) {
		t.Fatalf("source changed: %#v", src)
	}
}
