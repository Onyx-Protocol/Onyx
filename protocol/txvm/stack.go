package txvm

import "golang.org/x/crypto/sha3"

// These codes identify stacks.
// For example, ROLL reads a stack code
// to select which stack to modify.
const (
	StackData           = 0
	StackAlt            = 1
	StackInput          = 2
	StackValue          = 3
	StackOutput         = 4
	StackCond           = 5
	StackNonce          = 6
	StackAnchor         = 7
	StackRetirement     = 8
	StackTimeConstraint = 9
	StackAnnotation     = 10
	StackSummary        = 11
	NumStacks           = 12
)

var stackNames = map[string]int64{
	"datastack":           StackData,
	"altstack":            StackAlt,
	"inputstack":          StackInput,
	"valuestack":          StackValue,
	"outputstack":         StackOutput,
	"condstack":           StackCond,
	"noncestack":          StackNonce,
	"anchorstack":         StackAnchor,
	"retirementstack":     StackRetirement,
	"timeconstraintstack": StackTimeConstraint,
	"annotationstack":     StackAnnotation,
	"summarystack":        StackSummary,
}

type Stack interface {
	Len() int
	Element(n int) Value
	ID(n int) []byte
}

type stack struct {
	a []Value
}

func (s *stack) Len() int {
	return len(s.a)
}

func (s *stack) Element(n int) Value {
	return s.a[len(s.a)-1-n]
}

func (s *stack) Push(v Value) {
	s.a = append(s.a, v)
}

func (s *stack) PushBytes(b []byte) { s.Push(Bytes(b)) }
func (s *stack) PushInt64(n int64)  { s.Push(Int64(n)) }
func (s *stack) PushTuple(t Tuple)  { s.Push(t) }

func (s *stack) Pop() Value {
	v := s.a[len(s.a)-1]
	s.a = s.a[:len(s.a)-1]
	return v
}

func (s *stack) PopBytes() []byte { return []byte(s.Pop().(Bytes)) }
func (s *stack) PopInt64() int64  { return int64(s.Pop().(Int64)) }
func (s *stack) PopTuple() Tuple  { return s.Pop().(Tuple) }

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

func (s *stack) Reverse(n int64) {
	for i := 0; i < int(n)/2; i++ {
		a := len(s.a) - i - 1
		b := len(s.a) - int(n) + i
		s.a[a], s.a[b] = s.a[b], s.a[a]
	}
}

func (s *stack) ID(n int) []byte {
	panic("unsupported")
}

type tupleID struct {
	tuple Tuple
	id    []byte
}

type tupleStack struct {
	a []tupleID
}

func (s *tupleStack) Len() int {
	return len(s.a)
}

func (s *tupleStack) Element(n int) Value {
	return s.a[n].tuple
}

func (s *tupleStack) ID(n int) []byte {
	return s.a[n].id
}

func (s *tupleStack) Pop() Tuple {
	v := s.a[len(s.a)-1]
	s.a = s.a[:len(s.a)-1]
	return v.tuple
}

func (s *tupleStack) Push(v Tuple) {
	s.a = append(s.a, tupleID{
		tuple: v,
		id:    calcID(v),
	})
}

func (s *tupleStack) Peek() Tuple {
	return s.a[len(s.a)-1].tuple
}

func (s *tupleStack) Roll(n int64) {
	i := len(s.a) - int(n)
	x := s.a[i]
	s.a = append(s.a[:i], s.a[i+1:]...)
	s.a = append(s.a, x)
}

func (s *tupleStack) Bury(n int64) {
	x := s.a[len(s.a)-1]
	s.a = s.a[:len(s.a)-1]
	i := len(s.a) - int(n)
	s.a = append(append(append([]tupleID{}, s.a[:i]...), x), s.a[i:]...)
}

func (s *tupleStack) Reverse(n int64) {
	for i := 0; i < int(n)/2; i++ {
		a := len(s.a) - i - 1
		b := len(s.a) - int(n) + i
		s.a[a], s.a[b] = s.a[b], s.a[a]
	}
}

func calcID(v Tuple) []byte {
	h := sha3.New256()
	h.Write([]byte("txvm"))
	h.Write(encode(v))
	return h.Sum(nil)
}
