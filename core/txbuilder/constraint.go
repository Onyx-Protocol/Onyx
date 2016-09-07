package txbuilder

import (
	"encoding/json"

	chainjson "chain/encoding/json"
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
		case "ttl":
			var t struct {
				TTL int64
			}
			err = json.Unmarshal(p, &t)
			if err != nil {
				return err
			}
			constraint = TTLConstraint(t.TTL)
		case "outpoint":
			var t struct {
				bc.Outpoint
			}
			err = json.Unmarshal(p, &t)
			if err != nil {
				return err
			}
			constraint = OutpointConstraint(t.Outpoint)
		case "payment":
			var t PayConstraint
			err = json.Unmarshal(p, &t)
			if err != nil {
				return err
			}
			constraint = &t
		default:
			return errors.WithDetailf(ErrBadConstraint, "constraint %d has unknown type '%s'", i, t.Type)
		}
		*cl = append(*cl, constraint)
	}
	return nil
}

// TTLConstraint means the tx is only valid until the given time (in
// milliseconds since 1970).
type TTLConstraint int64

func (t TTLConstraint) Code() []byte {
	builder := vmutil.NewBuilder()
	builder.AddInt64(int64(t)).AddOp(vm.OP_MAXTIME).AddOp(vm.OP_LESSTHAN)
	return builder.Program
}

func (t TTLConstraint) MarshalJSON() ([]byte, error) {
	s := struct {
		Type string `json:"type"`
		TTL  int64  `json:"ttl"`
	}{
		Type: "ttl",
		TTL:  int64(t),
	}
	return json.Marshal(s)
}

// OutpointConstraint requires the outpoint being spent to equal the
// given value.
type OutpointConstraint bc.Outpoint

func (o OutpointConstraint) Code() []byte {
	builder := vmutil.NewBuilder()
	builder.AddData(o.Hash[:]).AddInt64(int64(o.Index))
	builder.AddOp(vm.OP_OUTPOINT)                     // stack is now [... hash index hash index]
	builder.AddOp(vm.OP_ROT)                          // stack is now [... hash hash index index]
	builder.AddOp(vm.OP_NUMEQUAL).AddOp(vm.OP_VERIFY) // stack is now [... hash hash]
	builder.AddOp(vm.OP_EQUAL)
	return builder.Program
}

func (o OutpointConstraint) MarshalJSON() ([]byte, error) {
	s := struct {
		Type string `json:"type"`
		bc.Outpoint
	}{
		Type:     "outpoint",
		Outpoint: bc.Outpoint(o),
	}
	return json.Marshal(s)
}

// PayConstraint requires the transaction to pay (at least) the given
// amount of the given asset to the given program, optionally with the
// given refdatahash.
type PayConstraint struct {
	bc.AssetAmount
	Program     chainjson.HexBytes `json:"program"`
	RefDataHash *bc.Hash           `json:"refdata_hash,omitempty"`
}

func (p PayConstraint) Code() []byte {
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

func (p PayConstraint) MarshalJSON() ([]byte, error) {
	s := struct {
		Type string `json:"type"`
		bc.AssetAmount
		Program     chainjson.HexBytes `json:"program"`
		RefDataHash *bc.Hash           `json:"refdata_hash,omitempty"`
	}{
		Type:        "payment",
		AssetAmount: p.AssetAmount,
		Program:     p.Program,
		RefDataHash: p.RefDataHash,
	}
	return json.Marshal(s)
}
