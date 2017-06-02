package data

type Value interface {
	value()
}

func (Bytes) value() {}
func (Int64) value() {}
func (List) value()  {}

// Bool converts x to a Value (either 0 or 1).
func Bool(x bool) Value {
	if x {
		return Int64(1)
	}
	return Int64(0)
}
