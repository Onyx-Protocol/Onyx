package txbuilder

import (
	"golang.org/x/crypto/sha3"

	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

// Constraint types express a constraint on an input of a proposed
// transaction, and know how to turn that constraint into part of a
// signature program in that input's witness.
type constraint interface {
	// Code produces bytecode expressing the constraint. The code, when
	// executed, must consume nothing from the stack and leave a new
	// boolean value on top of it.
	code() []byte
}

// timeConstraint means the tx is only valid within the given time
// bounds.  Either value is allowed to be 0 meaning "ignore."
type timeConstraint struct {
	minTimeMS, maxTimeMS uint64
}

func (t timeConstraint) code() []byte {
	if t.minTimeMS == 0 && t.maxTimeMS == 0 {
		return []byte{byte(vm.OP_TRUE)}
	}
	builder := vmutil.NewBuilder()
	if t.minTimeMS > 0 {
		builder.AddOp(vm.OP_MINTIME).AddInt64(int64(t.minTimeMS)).AddOp(vm.OP_GREATERTHANOREQUAL)
	}
	if t.maxTimeMS > 0 {
		if t.minTimeMS > 0 {
			// Consume the boolean left by the "mintime" clause, failing
			// immediately if it's false, so that the result of the
			// "maxtime" clause below is really (mintime clause && maxtime
			// clause).
			builder.AddOp(vm.OP_VERIFY)
		}
		builder.AddOp(vm.OP_MAXTIME).AddInt64(int64(t.maxTimeMS)).AddOp(vm.OP_LESSTHANOREQUAL)
	}
	return builder.Program
}

// outpointConstraint requires the outpoint being spent to equal the
// given value.
type outpointConstraint bc.Outpoint

func (o outpointConstraint) code() []byte {
	builder := vmutil.NewBuilder()
	builder.AddData(o.Hash[:]).AddInt64(int64(o.Index))
	builder.AddOp(vm.OP_OUTPOINT)                     // stack is now [... hash index hash index]
	builder.AddOp(vm.OP_ROT)                          // stack is now [... hash hash index index]
	builder.AddOp(vm.OP_NUMEQUAL).AddOp(vm.OP_VERIFY) // stack is now [... hash hash]
	builder.AddOp(vm.OP_EQUAL)
	return builder.Program
}

// refdataConstraint requires the input refdatahash to match that of
// the given data.
type refdataConstraint []byte

func (r refdataConstraint) code() []byte {
	h := sha3.Sum256(r)
	builder := vmutil.NewBuilder()
	builder.AddData(h[:]).AddOp(vm.OP_REFDATAHASH).AddOp(vm.OP_EQUAL)
	return builder.Program
}

// PayConstraint requires the transaction to pay (at least) the given
// amount of the given asset to the given program, optionally with the
// given refdatahash.
type payConstraint struct {
	bc.AssetAmount
	Program     []byte
	RefDataHash *bc.Hash
}

func (p payConstraint) code() []byte {
	builder := vmutil.NewBuilder()
	if p.RefDataHash == nil {
		builder.AddData([]byte{})
	} else {
		builder.AddData((*p.RefDataHash)[:])
	}
	builder.AddInt64(int64(p.Amount)).AddData(p.AssetID[:]).AddInt64(1).AddData(p.Program)
	builder.AddOp(vm.OP_FINDOUTPUT)
	return builder.Program
}
