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

func (s *stack) peek(n int64) (value, bool) {
	index := int64(len(*s)) - 1 - n
	if index < 0 || index >= int64(len(*s)) {
		return nil, false
	}
	return (*s)[index], true
}

func (s *stack) push(v value) {
	*s = append(*s, v)
}

func (s *stack) pop() (value, bool) {
	res, ok := s.peek(0)
	if ok {
		*s = (*s)[:len(*s)-1]
	}
	return res, ok
}

func (s *stack) pushN(vals []value) {
	*s = append(*s, vals...)
}

func (s *stack) popN(n int64) []value {
	var res []value
	for n > 0 && len(*s) > 0 {
		res = append(res, (*s)[len(*s)-1])
		*s = (*s)[:len(*s)-1]
		n--
	}
	return res
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
