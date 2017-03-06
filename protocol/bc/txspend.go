package bc

import (
	"chain/errors"
	"chain/protocol/vm"
)

// Spend accesses the value in a prior Output for transfer
// elsewhere. It satisfies the Entry interface.
//
// (Not to be confused with the deprecated type SpendInput.)
type Spend struct {
	Body struct {
		SpentOutputID Hash // the hash of an output entry
		Data          Hash
		ExtHash       Hash
	}
	ordinal int

	Witness struct {
		Destination ValueDestination
		Arguments   [][]byte
		AnchoredID  Hash
	}

	// SpentOutput contains (a pointer to) the manifested entry
	// corresponding to Body.SpentOutputID.
	SpentOutput *Output

	// Anchored contains a pointer to the manifested entry corresponding
	// to witness.AnchoredID.
	Anchored Entry
}

func (Spend) Type() string         { return "spend1" }
func (s *Spend) body() interface{} { return s.Body }

func (s Spend) Ordinal() int { return s.ordinal }

func (s *Spend) SetDestination(id Hash, val AssetAmount, pos uint64, e Entry) {
	s.Witness.Destination = ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
		Entry:    e,
	}
}

// NewSpend creates a new Spend.
func NewSpend(out *Output, data Hash, ordinal int) *Spend {
	s := new(Spend)
	s.Body.SpentOutputID = EntryID(out)
	s.Body.Data = data
	s.ordinal = ordinal
	s.SpentOutput = out
	return s
}

func (s *Spend) SetAnchored(id Hash, entry Entry) {
	s.Witness.AnchoredID = id
	s.Anchored = entry
}

func (s *Spend) CheckValid(vs *validationState) error {
	err := vm.Verify(NewTxVMContext(vs.tx, s, s.SpentOutput.Body.ControlProgram, s.Witness.Arguments))
	if err != nil {
		return errors.Wrap(err, "checking control program")
	}

	if s.SpentOutput.Body.Source.Value != s.Witness.Destination.Value {
		return errors.WithDetailf(
			errMismatchedValue,
			"previous output is for %d unit(s) of %x, spend wants %d unit(s) of %x",
			s.SpentOutput.Body.Source.Value.Amount,
			s.SpentOutput.Body.Source.Value.AssetID[:],
			s.Witness.Destination.Value.Amount,
			s.Witness.Destination.Value.AssetID[:],
		)
	}

	vs2 := *vs
	vs2.destPos = 0
	err = s.Witness.Destination.CheckValid(&vs2)
	if err != nil {
		return errors.Wrap(err, "checking spend destination")
	}

	if vs.tx.Body.Version == 1 && (s.Body.ExtHash != Hash{}) {
		return errNonemptyExtHash
	}

	return nil
}
