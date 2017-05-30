package compiler

type (
	stack struct {
		*stackEntry
	}
	stackEntry struct {
		str  string
		prev *stackEntry
	}
)

func (stk stack) isEmpty() bool {
	return stk.stackEntry == nil
}

func (stk stack) top() string {
	if stk.isEmpty() {
		return ""
	}
	return stk.str
}

func (stk stack) add(str string) stack {
	e := &stackEntry{
		str:  str,
		prev: stk.stackEntry,
	}
	return stack{e}
}

func (stk stack) addFromStack(other stack) stack {
	if other.isEmpty() {
		return stk
	}
	res := stk.addFromStack(other.drop())
	return res.add(other.top())
}

func (stk stack) drop() stack {
	if !stk.isEmpty() {
		stk = stack{stk.prev}
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
	if stk.isEmpty() {
		return -1
	}
	if stk.str == str {
		return 0
	}
	res := stk.drop().find(str)
	if res < 0 {
		return res
	}
	return res + 1
}

func (stk stack) roll(n int) stack {
	var x func(stack, int) (stack, string)
	x = func(stk stack, n int) (stack, string) {
		if n == 0 {
			return stk.drop(), stk.top()
		}
		stk2, entry := x(stk.drop(), n-1)
		return stk2.add(stk.top()), entry
	}
	stk, entry := x(stk, n)
	return stk.add(entry)
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

func (stk stack) over() stack {
	t := stk.drop().top()
	return stk.add(t)
}

func (stk stack) pick(n int) stack {
	t := stk.dropN(n).top()
	return stk.add(t)
}

func (stk stack) String() string {
	if stk.stackEntry == nil {
		return "[]"
	}
	var x func(stk stack) string
	x = func(stk stack) string {
		if stk.stackEntry == nil {
			return ""
		}
		return x(stk.drop()) + " " + stk.stackEntry.str
	}
	return "[..." + x(stk) + "]"
}
