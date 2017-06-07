package txvm

type Value interface {
	value()
}

// Bool converts x to a Value (either 0 or 1).
func Bool(x bool) Value {
	if x {
		return Int64(1)
	}
	return Int64(0)
}

type Bytes []byte

type Int64 int64

type VMTuple []Value

type stack struct {
	a []Value
}

func (s *stack) Len() int64 {
	return int64(len(s.a))
}

func (s *stack) Push(v Value) {
	s.a = append(s.a, v)
}

func (s *stack) PushBytes(b []byte) { s.Push(Bytes(b)) }
func (s *stack) PushInt64(n int64)  { s.Push(Int64(n)) }

func (s *stack) Pop() Value {
	v := s.a[len(s.a)-1]
	s.a = s.a[:len(s.a)-1]
	return v
}

func (s *stack) PopBytes() []byte { return []byte(s.Pop().(Bytes)) }
func (s *stack) PopInt64() int64  { return int64(s.Pop().(Int64)) }

func (s *stack) Roll(n int64) {
	i := len(s.a) - int(n)
	x := s.a[i]
	s.a = append(s.a[:i], s.a[i+1:]...)
	s.Push(x)
}

func (Bytes) value()   {}
func (Int64) value()   {}
func (VMTuple) value() {}
