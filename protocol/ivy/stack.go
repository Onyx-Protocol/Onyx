package ivy

type (
	stack      *stackEntry
	stackEntry struct {
		str  string
		prev *stackEntry
	}
)

func (stk stack) top() string {
	if stk == nil {
		return ""
	}
	return (*stackEntry)(stk).str
}

func (stk stack) add(str string) stack {
	e := &stackEntry{
		str:  str,
		prev: stk,
	}
	return stack(e)
}

func (stk stack) addFromStack(other stack) stack {
	if other == nil {
		return stk
	}
	res := stk.addFromStack(other.drop())
	return res.add(other.top())
}

func (stk stack) drop() stack {
	if stk != nil {
		stk = (*stackEntry)(stk).prev
	}
	return stk
}

func (stk stack) dropN(n int) stack {
	for n > 0 {
		stk = stk.drop()
		n--
	}
	return stk
}

func (stk stack) find(str string) int {
	if stk == nil {
		return -1
	}
	if (*stackEntry)(stk).str == str {
		return 0
	}
	res := stk.drop().find(str)
	if res < 0 {
		return res
	}
	return res + 1
}

func (stk stack) roll(n int) stk {
	if n == 0 {
		return stk
	}
	t := stk.top()
	stk = stk.roll(n - 1)
	return stk.add(t).swap()
}

func (stk stack) swap() stack {
	a := stk.top()
	stk = stk.drop()
	b := stk.top()
	stk = stk.drop()
	return stk.add(a).add(b)
}

func (stk stack) dup() stack {
	return stk.add(stk.top())
}
