package data

type List struct {
	a []Value
}

func (l *List) Len() int64 {
	return int64(len(l.a))
}

func (l *List) Push(v Value) {
	l.a = append(l.a, v)
}

func (l *List) PushBytes(b []byte) { l.Push(Bytes(b)) }
func (l *List) PushInt64(n int64)  { l.Push(Int64(n)) }

func (l *List) Pop() Value {
	v := l.a[len(l.a)-1]
	l.a = l.a[:len(l.a)-1]
	return v
}

func (l *List) PopBytes() []byte { return []byte(l.Pop().(Bytes)) }
func (l *List) PopInt64() int64  { return int64(l.Pop().(Int64)) }

func (l *List) Roll(n int64) {
	i := len(l.a) - int(n)
	x := l.a[i]
	l.a = append(l.a[:i], l.a[i+1:]...)
	l.Push(x)
}
