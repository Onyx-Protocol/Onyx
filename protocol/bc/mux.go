package bc

import (
	"chain/errors"
	"chain/math/checked"
	"chain/protocol/vm"
)

// Mux splits and combines value from one or more source entries,
// making it available to one or more destination entries. It
// satisfies the Entry interface.
type Mux struct {
	Body struct {
		Sources []ValueSource // issuances, spends, and muxes
		Program Program
		ExtHash Hash
	}

	Witness struct {
		Destinations []ValueDestination // outputs, retirements, and muxes
		Arguments    [][]byte
	}
}

func (Mux) Type() string         { return "mux1" }
func (m *Mux) body() interface{} { return m.Body }

func (Mux) Ordinal() int { return -1 }

// NewMux creates a new Mux.
func NewMux(sources []ValueSource, program Program) *Mux {
	m := new(Mux)
	m.Body.Sources = sources
	m.Body.Program = program
	return m
}

func (mux *Mux) CheckValid(vs *validationState) error {
	err := vm.Verify(NewTxVMContext(vs.tx, mux, mux.Body.Program, mux.Witness.Arguments))
	if err != nil {
		return errors.Wrap(err, "checking mux program")
	}

	for i, src := range mux.Body.Sources {
		vs2 := *vs
		vs2.sourcePos = uint64(i)
		err := src.CheckValid(&vs2)
		if err != nil {
			return errors.Wrapf(err, "checking mux source %d", i)
		}
	}

	for i, dest := range mux.Witness.Destinations {
		vs2 := *vs
		vs2.destPos = uint64(i)
		err := dest.CheckValid(&vs2)
		if err != nil {
			return errors.Wrapf(err, "checking mux destination %d", i)
		}
	}

	parity := make(map[AssetID]int64)
	for i, src := range mux.Body.Sources {
		sum, ok := checked.AddInt64(parity[src.Value.AssetID], int64(src.Value.Amount))
		if !ok {
			return errors.WithDetailf(errOverflow, "adding %d units of asset %x from mux source %d to total %d overflows int64", src.Value.Amount, src.Value.AssetID[:], i, parity[src.Value.AssetID])
		}
		parity[src.Value.AssetID] = sum
	}

	for i, dest := range mux.Witness.Destinations {
		sum, ok := parity[dest.Value.AssetID]
		if !ok {
			return errors.WithDetailf(errNoSource, "mux destination %d, asset %x, has no corresponding source", i, dest.Value.AssetID[:])
		}

		diff, ok := checked.SubInt64(sum, int64(dest.Value.Amount))
		if !ok {
			return errors.WithDetailf(errOverflow, "subtracting %d units of asset %x from mux destination %d from total %d underflows int64", dest.Value.Amount, dest.Value.AssetID[:], i, sum)
		}
		parity[dest.Value.AssetID] = diff
	}

	for assetID, amount := range parity {
		if amount != 0 {
			return errors.WithDetailf(errUnbalanced, "asset %x sources - destinations = %d (should be 0)", assetID[:], amount)
		}
	}

	if vs.tx.Body.Version == 1 && (mux.Body.ExtHash != Hash{}) {
		return errNonemptyExtHash
	}

	return nil
}
