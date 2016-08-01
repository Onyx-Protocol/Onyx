package accounts

import (
	"reflect"
	"testing"
)

func TestBreakupChange(t *testing.T) {
	got := breakupChange(1)
	want := []uint64{1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}

	// Now do a lot of BreakupChange calls and expect that at least one
	// of them will create two or more pieces.  Check that in all cases
	// the pieces add up to the input.
	var anyMultiples bool
	for i := 0; i < 100; i++ {
		got := breakupChange(100)
		var sum uint64
		for _, n := range got {
			sum += n
		}
		if sum != 100 {
			t.Errorf("sum of %v is %d, not 100", got, sum)
		}
		if len(got) > 1 {
			anyMultiples = true
		}
	}

	if !anyMultiples {
		t.Errorf("no calls produced multiple change pieces, that's very unlikely")
	}
}
