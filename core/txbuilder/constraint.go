package txbuilder

import (
	"encoding/json"

	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

// Constraint types express a constraint on an input of a proposed
// transaction, and know how to turn that constraint into part of a
// p2dp program in that input's witness.
type Constraint interface {
	// Code produces bytecode expressing the constraint. The code, when
	// executed, must consume nothing from the stack and leave a new
	// boolean value on top of it.
	Code() []byte
}

// TxHashConstraint requires the transaction to have the given hash.
type TxHashConstraint bc.Hash

func (t TxHashConstraint) Code() []byte {
	builder := vmutil.NewBuilder()
	builder.AddData(t[:])
	builder.AddInt64(1).AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
	return builder.Program
}

func (t TxHashConstraint) MarshalJSON() ([]byte, error) {
	s := struct {
		Type string  `json:"type"`
		Hash bc.Hash `json:"transaction_id"`
	}{
		Type: "transaction_id",
		Hash: bc.Hash(t),
	}
	return json.Marshal(s)
}
