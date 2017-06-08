package txvm

const (
	TypeInt64  = 0
	TypeString = 1
	TypeTuple  = 2
)

type Value interface {
	value()
	typ() int
}

// Bool converts x to a Value (either 0 or 1).
func Bool(x bool) Value {
	if x {
		return Int64(1)
	}
	return Int64(0)
}

// toBool converts v from a Value to a bool
func toBool(v Value) bool {
	n, ok := v.(Int64)
	return !ok || n != 0
}

type Bytes []byte

type Int64 int64

type VMTuple []Value

func (Bytes) value()   {}
func (Int64) value()   {}
func (VMTuple) value() {}

func (Bytes) typ() int   { return TypeString }
func (Int64) typ() int   { return TypeInt64 }
func (VMTuple) typ() int { return TypeTuple }
