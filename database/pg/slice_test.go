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

func TestSliceNullStringsValue(t *testing.T) {
	tests := []struct {
		in  NullStrings
		out string
	}{
		{NullStrings{{"a", true}, {"b", true}, {"c", true}}, `{"a","b","c"}`},
		{NullStrings{{"a", true}, {"b", false}, {"c", true}}, `{"a",NULL,"c"}`},
		{NullStrings{{"", false}, {"NULL", false}, {"NULL", true}}, `{NULL,NULL,"NULL"}`},
		{NullStrings{{"a b", true}, {"c'd", true}}, `{"a b","c'd"}`},
		{NullStrings{{`a"`, true}, {"this,can,handle,commas", true}}, `{"a\"","this,can,handle,commas"}`},
	}

	for i, test := range tests {
		val, err := test.in.Value()
		if err != nil {
			t.Errorf("%d: unexpected error: %s", i, err)
			continue
		}
		b, ok := val.([]byte)
		if !ok {
			t.Errorf("%d: could not type assert to []byte", i)
			continue
		}
		if !reflect.DeepEqual(string(b), test.out) {
			t.Errorf("%d: Scan(%v) got %v want %v", i, test.in, string(b), test.out)
		}
	}
}

func TestSliceStringScanErr(t *testing.T) {
	s := `{","}`
	var x Strings
	err := x.Scan([]byte(s))
	if err == nil {
		t.Errorf("Scan(%#q) = nil want error", s)
	}
}

func TestSliceByteaValue(t *testing.T) {
	v := Byteas{
		[]byte("foo"),
		[]byte("bar"),
	}

	const want = `{\\x666f6f,\\x626172}`

	got, err := v.Value()
	if err != nil {
		t.Fatal(err)
	}
	if s := string(got.([]byte)); s != want {
		t.Errorf("%v.Value() got %#q want %#q", v, s, want)
	}
}

func TestBoolsScan(t *testing.T) {
	tests := []struct {
		in  string
		out Bools
	}{
		{`{t,f}`, Bools{true, false}},
		{`{f}`, Bools{false}},
		{`{}`, nil},
	}
	for i, test := range tests {
		var arr Bools
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

func TestBoolsValue(t *testing.T) {
	tests := []struct {
		in  Bools
		out string
	}{
		{Bools{true, false}, `{t,f}`},
		{Bools{false}, `{f}`},
		{Bools{}, `{}`},
		{nil, `{}`},
	}

	for i, test := range tests {
		val, err := test.in.Value()
		if err != nil {
			t.Errorf("%d: unexpected error: %s", i, err)
			continue
		}
		b, ok := val.([]byte)
		if !ok {
			t.Errorf("%d: could not type assert to []byte", i)
			continue
		}
		if !reflect.DeepEqual(string(b), test.out) {
			t.Errorf("%d: Scan(%v) got %v want %v", i, test.in, string(b), test.out)
		}
	}
}
