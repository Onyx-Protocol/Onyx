package txvm

const (
	datastack    int64 = 0
	altstack     int64 = 1
	entrystack   int64 = 2
	commandstack int64 = 3
	effectstack  int64 = 4

	numstacks = effectstack + 1
)

type stack []Item

func (s *stack) peek(n int64) (Item, bool) {
	index := int64(len(*s)) - 1 - n
	if index < 0 || index >= int64(len(*s)) {
		return nil, false
	}
	return (*s)[index], true
}

func (s *stack) push(v Item) {
	*s = append(*s, v)
}

func (s *stack) pop() (Item, bool) {
	res, ok := s.peek(0)
	if ok {
		*s = (*s)[:len(*s)-1]
	}
	return res, ok
}

func (s *stack) pushN(vals []Item) {
	*s = append(*s, vals...)
}

func (s *stack) popN(n int64) []Item {
	var res []Item
	for n > 0 && len(*s) > 0 {
		res = append(res, (*s)[len(*s)-1])
		*s = (*s)[:len(*s)-1]
		n--
	}
	return res
}

func (s *stack) peekN(n int64) []Item {
	var res []Item
	for n > 0 && n < int64(len(*s)) {
		res = append(res, (*s)[int64(len(*s))-1-n])
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

type Stack interface {
	Len() int
	Element(n int) Item
}

func (s *stack) Len() int {
	return len(*s)
}

func (s *stack) Element(n int) Item {
	return (*s)[len(*s)-1-n]
}
