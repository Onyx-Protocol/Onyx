package txvm2

const (
	datastack    = 0
	altstack     = 1
	entrystack   = 2
	commandstack = 3
	effectstack  = 4

	numstacks = effectstack + 1
)

type stack []value

func (s *stack) peek() (value, bool) {
	if len(s) == 0 {
		return nil, false
	}
	return s[len(s)-1], true
}

func (s *stack) push(v value) {
	*s = append(*s, v)
}

func (s *stack) pop() (value, bool) {
	res, ok := s.peek()
	if ok {
		*s = s[:len(s)-1]
	}
	return res, ok
}

func (s *stack) isEmpty() bool {
	return len(s) == 0
}
