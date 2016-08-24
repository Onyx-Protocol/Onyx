package main

import (
	"strings"

	"golang.org/x/crypto/sha3"

	"chain/protocol/bc"
	"chain/protocol/vm"
)

type (
	step struct {
		ops   string
		stack stack
	}

	translation struct {
		steps []step          // computed by translatable.translate()
		ops   string          // computed and memoized from steps
		bytes []byte          // computed and memoized from ops
		hash  bc.ContractHash // computed and memoized from bytes
	}

	translatable interface {
		// The translate() function takes a representation of the stack as
		// an argument, for resolving variable references.  stack[0] is
		// the top of the stack.  It also takes a "context" containing all
		// known parsed contracts, for resolving contract calls.
		//
		// Statements, decls, and exprs are all translatable.  Statements
		// produce code that leaves the stack unchanged.  Decls add one
		// named item to the top of the stack (for the scope of the clause
		// in which they appear).  Exprs add one unnamed item to the top
		// of the stack.
		translate(stack, *context) (*translation, error)
	}
)

func (t translation) finalStack() stack {
	return t.steps[len(t.steps)-1].stack
}

func (t translation) finalStackTop() typedName {
	return t.finalStack().top()
}

// add (and addMany below) can take a nil receiver for the purpose of
// building up translations from scratch.
func (t *translation) add(ops string, stk stack) *translation {
	if t == nil {
		return &translation{steps: []step{{ops, stk}}}
	}
	return &translation{steps: append(t.steps, step{ops, stk})}
}

func (t *translation) addMany(steps []step) *translation {
	if t == nil {
		return &translation{steps: steps}
	}
	return &translation{steps: append(t.steps, steps...)}
}

func (t translation) getOps() string {
	if t.ops == "" {
		o := make([]string, 0, len(t.steps))
		for _, step := range t.steps {
			o = append(o, step.ops)
		}
		t.ops = strings.Join(o, " ")
	}
	return t.ops
}

func (t translation) getBytes() ([]byte, error) {
	if t.bytes == nil {
		b, err := vm.Compile(t.getOps())
		if err != nil {
			return nil, err
		}
		t.bytes = b
	}
	return t.bytes, nil
}

func (t translation) getHash() (bc.ContractHash, error) {
	var zeroHash bc.ContractHash

	if t.hash == zeroHash {
		b, err := t.getBytes()
		if err != nil {
			return zeroHash, err
		}
		h := sha3.Sum256(b)
		copy(t.hash[:], h[:])
	}
	return t.hash, nil
}

// longestStep returns the length of the longest step.ops in t
func (t translation) opsWidth() (res int) {
	for _, s := range t.steps {
		if len(s.ops) > res {
			res = len(s.ops)
		}
	}
	return res
}

// stackStr produces a string depiction of the stack at the given step
func (s step) stackStr() string {
	strs := make([]string, 1, 1+len(s.stack))
	strs[0] = "<top>"
	for _, item := range s.stack {
		strs = append(strs, item.name)
	}
	return strings.Join(strs, " ")
}
