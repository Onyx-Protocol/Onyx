package checked

import (
	"math"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestInt64(t *testing.T) {
	cases := []struct {
		f          func(a, b int64) (int64, bool)
		a, b, want int64
		wantOk     bool
	}{
		{AddInt64, 2, 3, 5, true},
		{AddInt64, 2, -3, -1, true},
		{AddInt64, -2, -3, -5, true},
		{AddInt64, math.MaxInt64, 1, 0, false},
		{AddInt64, math.MinInt64, math.MinInt64, 0, false},
		{AddInt64, math.MinInt64, -1, 0, false},
		{SubInt64, 3, 2, 1, true},
		{SubInt64, 2, 3, -1, true},
		{SubInt64, -2, -3, 1, true},
		{SubInt64, math.MinInt64, 1, 0, false},
		{SubInt64, -2, math.MaxInt64, 0, false},
		{MulInt64, 2, 3, 6, true},
		{MulInt64, -2, -3, 6, true},
		{MulInt64, -2, 3, -6, true},
		{MulInt64, math.MaxInt64, -1, math.MinInt64 + 1, true},
		{MulInt64, math.MinInt64, 2, 0, false},
		{MulInt64, math.MaxInt64, 2, 0, false},
		{MulInt64, 2, math.MinInt64, 0, false},
		{MulInt64, -2, math.MinInt64, 0, false},
		{DivInt64, 2, 2, 1, true},
		{DivInt64, -2, -2, 1, true},
		{DivInt64, -2, 2, -1, true},
		{DivInt64, 1, 0, 0, false},
		{DivInt64, math.MinInt64, -1, 0, false},
		{ModInt64, 3, 2, 1, true},
		{ModInt64, -3, -2, -1, true},
		{ModInt64, -3, 2, -1, true},
		{ModInt64, 1, 0, 0, false},
		{ModInt64, math.MinInt64, -1, 0, false},
		{LshiftInt64, 1, 2, 4, true},
		{LshiftInt64, -1, 2, -4, true},
		{LshiftInt64, 1, 64, 0, false},
		{LshiftInt64, 2, 63, 0, false},
	}

	for _, c := range cases {
		got, gotOk := c.f(c.a, c.b)

		if got != c.want {
			t.Errorf("%s(%d, %d) = %d want %d", fname(c.f), c.a, c.b, got, c.want)
		}

		if gotOk != c.wantOk {
			t.Errorf("%s(%d, %d) ok = %v want %v", fname(c.f), c.a, c.b, gotOk, c.wantOk)
		}
	}

	negateCases := []struct {
		a, want int64
		wantOk  bool
	}{
		{1, -1, true},
		{-1, 1, true},
		{0, 0, true},
		{math.MinInt64, 0, false},
	}
	for _, c := range negateCases {
		got, gotOk := NegateInt64(c.a)

		if got != c.want {
			t.Errorf("NegateInt64(%d) = %d want %d", c.a, got, c.want)
		}

		if gotOk != c.wantOk {
			t.Errorf("NegateInt64(%d) ok = %v want %v", c.a, gotOk, c.wantOk)
		}
	}
}

func TestUint64(t *testing.T) {
	cases := []struct {
		f          func(a, b uint64) (uint64, bool)
		a, b, want uint64
		wantOk     bool
	}{
		{AddUint64, 2, 3, 5, true},
		{AddUint64, math.MaxUint64, 1, 0, false},
		{SubUint64, 3, 2, 1, true},
		{SubUint64, 2, 3, 0, false},
		{MulUint64, 2, 3, 6, true},
		{MulUint64, math.MaxUint64, 2, 0, false},
		{DivUint64, 2, 2, 1, true},
		{DivUint64, 1, 0, 0, false},
		{ModUint64, 3, 2, 1, true},
		{ModUint64, 1, 0, 0, false},
		{LshiftUint64, 1, 2, 4, true},
		{LshiftUint64, 1, 64, 0, false},
		{LshiftUint64, 2, 63, 0, false},
	}

	for _, c := range cases {
		got, gotOk := c.f(c.a, c.b)

		if got != c.want {
			t.Errorf("%s(%d, %d) = %d want %d", fname(c.f), c.a, c.b, got, c.want)
		}

		if gotOk != c.wantOk {
			t.Errorf("%s(%d, %d) ok = %v want %v", fname(c.f), c.a, c.b, gotOk, c.wantOk)
		}
	}
}

func TestInt32(t *testing.T) {
	cases := []struct {
		f          func(a, b int32) (int32, bool)
		a, b, want int32
		wantOk     bool
	}{
		{AddInt32, 2, 3, 5, true},
		{AddInt32, 2, -3, -1, true},
		{AddInt32, -2, -3, -5, true},
		{AddInt32, math.MaxInt32, 1, 0, false},
		{AddInt32, math.MinInt32, math.MinInt32, 0, false},
		{AddInt32, math.MinInt32, -1, 0, false},
		{SubInt32, 3, 2, 1, true},
		{SubInt32, 2, 3, -1, true},
		{SubInt32, -2, -3, 1, true},
		{SubInt32, math.MinInt32, 1, 0, false},
		{SubInt32, -2, math.MaxInt32, 0, false},
		{MulInt32, 2, 3, 6, true},
		{MulInt32, -2, -3, 6, true},
		{MulInt32, -2, 3, -6, true},
		{MulInt32, math.MaxInt32, -1, math.MinInt32 + 1, true},
		{MulInt32, math.MinInt32, 2, 0, false},
		{MulInt32, math.MaxInt32, 2, 0, false},
		{MulInt32, 2, math.MinInt32, 0, false},
		{MulInt32, -2, math.MinInt32, 0, false},
		{DivInt32, 2, 2, 1, true},
		{DivInt32, -2, -2, 1, true},
		{DivInt32, -2, 2, -1, true},
		{DivInt32, 1, 0, 0, false},
		{DivInt32, math.MinInt32, -1, 0, false},
		{ModInt32, 3, 2, 1, true},
		{ModInt32, -3, -2, -1, true},
		{ModInt32, -3, 2, -1, true},
		{ModInt32, 1, 0, 0, false},
		{ModInt32, math.MinInt32, -1, 0, false},
		{LshiftInt32, 1, 2, 4, true},
		{LshiftInt32, -1, 2, -4, true},
		{LshiftInt32, 1, 32, 0, false},
		{LshiftInt32, 2, 31, 0, false},
	}

	for _, c := range cases {
		got, gotOk := c.f(c.a, c.b)

		if got != c.want {
			t.Errorf("%s(%d, %d) = %d want %d", fname(c.f), c.a, c.b, got, c.want)
		}

		if gotOk != c.wantOk {
			t.Errorf("%s(%d, %d) ok = %v want %v", fname(c.f), c.a, c.b, gotOk, c.wantOk)
		}
	}

	negateCases := []struct {
		a, want int32
		wantOk  bool
	}{
		{1, -1, true},
		{-1, 1, true},
		{0, 0, true},
		{math.MinInt32, 0, false},
	}
	for _, c := range negateCases {
		got, gotOk := NegateInt32(c.a)

		if got != c.want {
			t.Errorf("NegateInt32(%d) = %d want %d", c.a, got, c.want)
		}

		if gotOk != c.wantOk {
			t.Errorf("NegateInt32(%d) ok = %v want %v", c.a, gotOk, c.wantOk)
		}
	}
}

func TestUint32(t *testing.T) {
	cases := []struct {
		f          func(a, b uint32) (uint32, bool)
		a, b, want uint32
		wantOk     bool
	}{
		{AddUint32, 2, 3, 5, true},
		{AddUint32, math.MaxUint32, 1, 0, false},
		{SubUint32, 3, 2, 1, true},
		{SubUint32, 2, 3, 0, false},
		{MulUint32, 2, 3, 6, true},
		{MulUint32, math.MaxUint32, 2, 0, false},
		{DivUint32, 2, 2, 1, true},
		{DivUint32, 1, 0, 0, false},
		{ModUint32, 3, 2, 1, true},
		{ModUint32, 1, 0, 0, false},
		{LshiftUint32, 1, 2, 4, true},
		{LshiftUint32, 1, 32, 0, false},
		{LshiftUint32, 2, 31, 0, false},
	}

	for _, c := range cases {
		got, gotOk := c.f(c.a, c.b)

		if got != c.want {
			t.Errorf("%s(%d, %d) = %d want %d", fname(c.f), c.a, c.b, got, c.want)
		}

		if gotOk != c.wantOk {
			t.Errorf("%s(%d, %d) ok = %v want %v", fname(c.f), c.a, c.b, gotOk, c.wantOk)
		}
	}
}

func fname(f interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	return name[strings.IndexRune(name, '.')+1:]
}
