package pg

import (
	"reflect"
	"testing"
)

func TestSliceStringScan(t *testing.T) {
	tests := []struct {
		in  string
		out Strings
	}{
		{`{"a","b","c"}`, Strings{"a", "b", "c"}},
		{`{a,b,c}`, Strings{"a", "b", "c"}},
		{`{"a b","c'd"}`, Strings{"a b", "c'd"}},
	}

	for i, test := range tests {
		var arr Strings
		err := arr.Scan([]byte(test.in))
		if err != nil {
			t.Errorf("%d: unexpected error: %s", i, err)
			continue
		}
		if !reflect.DeepEqual(arr, test.out) {
			t.Errorf("%d: Scan(%v) got %v want %v", i, test.in, arr, test.out)
		}
	}
}

func TestSliceStringValue(t *testing.T) {
	tests := []struct {
		in  Strings
		out string
	}{
		{Strings{"a", "b", "c"}, `{"a","b","c"}`},
		{Strings{"a b", "c'd"}, `{"a b","c'd"}`},
		{Strings{`a"`, "this,can,handle,commas"}, `{"a\"","this,can,handle,commas"}`},
	}

	for i, test := range tests {
		val, err := test.in.Value()
		if err != nil {
			t.Errorf("%d: unexpected error: %s", i, err)
			continue
		}
		b, ok := val.([]byte)
		if !ok {
			t.Errorf("%d: could not type asser to []byte", i)
			continue
		}
		if !reflect.DeepEqual(string(b), test.out) {
			t.Errorf("%d: Scan(%v) got %v want %v", i, test.in, string(b), test.out)
		}
	}
}
