package main

// Entries in a stack are typedName pairs.  If the name is a valid
// identifier, it can be found via lookup().  Values that aren't meant
// to be found via lookup() (such as intermediate expression values)
// are present both as placeholders, to maintain the proper stack
// depth when looking up things that _are_ identifiers, and to depict
// for human readers the progression of stack changes through the
// steps of evaluating a contract.  These items should not be valid
// identifiers.  This implementation uses [square brackets] to enclose
// such values.
type stack []typedName

func (s stack) top() typedName {
	return s[0]
}

func (s stack) drop() stack {
	return s.dropN(1)
}

func (s stack) dropN(n int) stack {
	return s[n:]
}

func (s stack) nip() stack {
	res := s[:1]
	res = append(res, s[2:]...)
	return res
}

func (s stack) push(item typedName) stack {
	res := make([]typedName, 1, 1+len(s))
	res[0] = item
	res = append(res, s...)
	return res
}

func (s stack) bottomAdd(item typedName) stack {
	return append(s, item)
}

func (s stack) bottomAddMany(items []typedName) stack {
	return append(s, items...)
}

func (s stack) lookup(name string) int {
	for i, entry := range s {
		if name == entry.name {
			return i
		}
	}
	return -1
}

func (s typedName) String() string {
	return s.name
}
