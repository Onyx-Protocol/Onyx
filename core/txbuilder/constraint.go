package txbuilder

import (
	"encoding/json"

	"chain/errors"
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

type ConstraintList []Constraint

func (cl *ConstraintList) UnmarshalJSON(b []byte) error {
	var pre []json.RawMessage
	err := json.Unmarshal(b, &pre)
	if err != nil {
		return err
	}
	for i, p := range pre {
		var t struct {
			Type string
		}
		err = json.Unmarshal(p, &t)
		if err != nil {
			return err
		}
		var constraint Constraint
		switch t.Type {
		case "transaction_id":
			var txhash struct {
				Hash bc.Hash `json:"transaction_id"`
			}
			err = json.Unmarshal(p, &txhash)
			if err != nil {
				return err
			}
			constraint = TxHashConstraint(txhash.Hash)
		default:
			return errors.WithDetailf(ErrBadConstraint, "constraint %d has unknown type '%s'", i, t.Type)
		}
		*cl = append(*cl, constraint)
	}
	return nil
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
