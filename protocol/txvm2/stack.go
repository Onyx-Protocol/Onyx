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
	if len(*s) == 0 {
		return nil, false
	}
	return (*s)[len(*s)-1], true
}

func (s *stack) push(v value) {
	*s = append(*s, v)
}

func (s *stack) pop() (value, bool) {
	res, ok := s.peek()
	if ok {
		*s = (*s)[:len(*s)-1]
	}
	return res, ok
}

func (s *stack) isEmpty() bool {
	return len(*s) == 0
}

// xxx range checking etc. for the following
func (s *stack) roll(n int64) {
	item := (*s)[n]
	*s = append((*s)[:n], (*s)[n+1:]...)
	*s = append(*s, item)
}

func (s *stack) bury(n int64) {
	item := (*s)[len(*s)-1]
	before := (*s)[:n]
	after := (*s)[n : len(*s)-1]
	*s = append(before, item)
	*s = append(*s, after...)
}
