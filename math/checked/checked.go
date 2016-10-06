/*
Package checked implements basic arithmetic operations
with underflow and overflow checks.
*/
package checked

import (
	"errors"
	"math"
)

var ErrOverflow = errors.New("arithmetic overflow")

// AddInt64 returns a + b
// with an integer overflow check.
func AddInt64(a, b int64) (sum int64, ok bool) {
	if (b > 0 && a > math.MaxInt64-b) ||
		(b < 0 && a < math.MinInt64-b) {
		return 0, false
	}
	return a + b, true
}

// SubInt64 returns a - b
// with an integer overflow check.
func SubInt64(a, b int64) (diff int64, ok bool) {
	if (b > 0 && a < math.MinInt64+b) ||
		(b < 0 && a > math.MaxInt64+b) {
		return 0, false
	}
	return a - b, true
}

// MulInt64 returns a * b
// with an integer overflow check.
func MulInt64(a, b int64) (product int64, ok bool) {
	if (a > 0 && b > 0 && a > math.MaxInt64/b) ||
		(a > 0 && b <= 0 && b < math.MinInt64/a) ||
		(a <= 0 && b > 0 && a < math.MinInt64/b) ||
		(a < 0 && b <= 0 && b < math.MaxInt64/a) {
		return 0, false
	}
	return a * b, true
}

// DivInt64 returns a / b
// with an integer overflow check.
func DivInt64(a, b int64) (quotient int64, ok bool) {
	if b == 0 || (a == math.MinInt64 && b == -1) {
		return 0, false
	}
	return a / b, true
}

// ModInt64 returns a % b
// with an integer overflow check.
func ModInt64(a, b int64) (remainder int64, ok bool) {
	if b == 0 || (a == math.MinInt64 && b == -1) {
		return 0, false
	}
	return a % b, true
}

// NegateInt64 returns -a
// with an integer overflow check.
func NegateInt64(a int64) (negated int64, ok bool) {
	if a == math.MinInt64 {
		return 0, false
	}
	return -a, true
}

// LshiftInt64 returns a << b
// with an integer overflow check.
func LshiftInt64(a, b int64) (result int64, ok bool) {
	if b < 0 || b >= 64 {
		return 0, false
	}
	if (a >= 0 && a > math.MaxInt64>>uint(b)) || (a < 0 && a < math.MinInt64>>uint(b)) {
		return 0, false
	}
	return a << uint(b), true
}

// AddInt32 returns a + b
// with an integer overflow check.
func AddInt32(a, b int32) (sum int32, ok bool) {
	if (b > 0 && a > math.MaxInt32-b) ||
		(b < 0 && a < math.MinInt32-b) {
		return 0, false
	}
	return a + b, true
}

// SubInt32 returns a - b
// with an integer overflow check.
func SubInt32(a, b int32) (diff int32, ok bool) {
	if (b > 0 && a < math.MinInt32+b) ||
		(b < 0 && a > math.MaxInt32+b) {
		return 0, false
	}
	return a - b, true
}

// MulInt32 returns a * b
// with an integer overflow check.
func MulInt32(a, b int32) (product int32, ok bool) {
	if (a > 0 && b > 0 && a > math.MaxInt32/b) ||
		(a > 0 && b <= 0 && b < math.MinInt32/a) ||
		(a <= 0 && b > 0 && a < math.MinInt32/b) ||
		(a < 0 && b <= 0 && b < math.MaxInt32/a) {
		return 0, false
	}
	return a * b, true
}

// DivInt32 returns a / b
// with an integer overflow check.
func DivInt32(a, b int32) (quotient int32, ok bool) {
	if b == 0 || (a == math.MinInt32 && b == -1) {
		return 0, false
	}
	return a / b, true
}

// ModInt32 returns a % b
// with an integer overflow check.
func ModInt32(a, b int32) (remainder int32, ok bool) {
	if b == 0 || (a == math.MinInt32 && b == -1) {
		return 0, false
	}
	return a % b, true
}

// NegateInt32 returns -a
// with an integer overflow check.
func NegateInt32(a int32) (negated int32, ok bool) {
	if a == math.MinInt32 {
		return 0, false
	}
	return -a, true
}

// LshiftInt32 returns a << b
// with an integer overflow check.
func LshiftInt32(a, b int32) (result int32, ok bool) {
	if b < 0 || b >= 32 {
		return 0, false
	}
	if (a >= 0 && a > math.MaxInt32>>uint(b)) || (a < 0 && a < math.MinInt32>>uint(b)) {
		return 0, false
	}
	return a << uint(b), true
}

// AddUint64 returns a + b
// with an integer overflow check.
func AddUint64(a, b uint64) (sum uint64, ok bool) {
	if math.MaxUint64-a < b {
		return 0, false
	}
	return a + b, true
}

// SubUint64 returns a - b
// with an integer overflow check.
func SubUint64(a, b uint64) (diff uint64, ok bool) {
	if a < b {
		return 0, false
	}
	return a - b, true
}

// MulUint64 returns a * b
// with an integer overflow check.
func MulUint64(a, b uint64) (product uint64, ok bool) {
	if b > 0 && a > math.MaxUint64/b {
		return 0, false
	}
	return a * b, true
}

// DivUint64 returns a / b
// with an integer overflow check.
func DivUint64(a, b uint64) (quotient uint64, ok bool) {
	if b == 0 {
		return 0, false
	}
	return a / b, true
}

// ModUint64 returns a % b
// with an integer overflow check.
func ModUint64(a, b uint64) (remainder uint64, ok bool) {
	if b == 0 {
		return 0, false
	}
	return a % b, true
}

// LshiftUint64 returns a << b
// with an integer overflow check.
func LshiftUint64(a, b uint64) (result uint64, ok bool) {
	if b >= 64 {
		return 0, false
	}
	if a > math.MaxUint64>>uint(b) {
		return 0, false
	}
	return a << uint(b), true
}

// AddUint32 returns a + b
// with an integer overflow check.
func AddUint32(a, b uint32) (sum uint32, ok bool) {
	if math.MaxUint32-a < b {
		return 0, false
	}
	return a + b, true
}

// SubUint32 returns a - b
// with an integer overflow check.
func SubUint32(a, b uint32) (diff uint32, ok bool) {
	if a < b {
		return 0, false
	}
	return a - b, true
}

// MulUint32 returns a * b
// with an integer overflow check.
func MulUint32(a, b uint32) (product uint32, ok bool) {
	if b > 0 && a > math.MaxUint32/b {
		return 0, false
	}
	return a * b, true
}

// DivUint32 returns a / b
// with an integer overflow check.
func DivUint32(a, b uint32) (quotient uint32, ok bool) {
	if b == 0 {
		return 0, false
	}
	return a / b, true
}

// ModUint32 returns a % b
// with an integer overflow check.
func ModUint32(a, b uint32) (remainder uint32, ok bool) {
	if b == 0 {
		return 0, false
	}
	return a % b, true
}

// LshiftUint32 returns a << b
// with an integer overflow check.
func LshiftUint32(a, b uint32) (result uint32, ok bool) {
	if b >= 32 {
		return 0, false
	}
	if a > math.MaxUint32>>uint(b) {
		return 0, false
	}
	return a << uint(b), true
}
