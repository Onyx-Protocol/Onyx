package txvm

// These codes identify stacks.
// For example, ROLL reads a stack code
// to select which stack to modify.
const (
	StackData   = 0
	StackAlt    = 1
	StackInput  = 2
	StackValue  = 3
	StackCond   = 4
	StackOutput = 5
	StackNonce  = 6
	StackAnchor = 7
	StackRetire = 8

	StackVM1Value  = 9
	StackVM1Mux    = 10
	StackVM1Nonce  = 11
	StackVM1Anchor = 12
)

type stack struct {
	a []Value
}

func (s *stack) Len() int64 {
	return int64(len(s.a))
}

func (s *stack) Push(v Value) {
	s.a = append(s.a, v)
}

func (s *stack) PushBytes(b []byte)  { s.Push(Bytes(b)) }
func (s *stack) PushInt64(n int64)   { s.Push(Int64(n)) }
func (s *stack) PushTuple(t []Value) { s.Push(VMTuple(t)) }

func (s *stack) Pop() Value {
	v := s.a[len(s.a)-1]
	s.a = s.a[:len(s.a)-1]
	return v
}

func (s *stack) PopBytes() []byte  { return []byte(s.Pop().(Bytes)) }
func (s *stack) PopInt64() int64   { return int64(s.Pop().(Int64)) }
func (s *stack) PopTuple() []Value { return []Value(s.Pop().(VMTuple)) }

func (s *stack) Peek() Value {
	return s.a[len(s.a)-1]
}

func (s *stack) Roll(n int64) {
	i := len(s.a) - int(n)
	x := s.a[i]
	s.a = append(s.a[:i], s.a[i+1:]...)
	s.Push(x)
}

func (s *stack) Bury(n int64) {
	x := s.Pop()
	i := len(s.a) - int(n)
	s.a = append(append(append([]Value{}, s.a[:i]...), x), s.a[i:]...)
}
