package bc

import (
	"fmt"

	"chain/errors"
)

type ValueSource struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64

	// The Entry corresponding to Ref, if available.  Under normal
	// conditions, this field is always set except when the containing
	// entry is an Output inside of a Spend (i.e., a prevout).
	//
	// The struct tag excludes the field from hashing.
	Entry `entry:"-"`
}

func (vs *ValueSource) CheckValid(vstate *validationState) error {
	vstate2 := *vstate
	vstate2.entryID = vs.Ref
	err := vs.Entry.CheckValid(&vstate2)
	if err != nil {
		return errors.Wrap(err, "checking value source")
	}

	var dest ValueDestination
	switch ref := vs.Entry.(type) {
	case *Issuance:
		if vs.Position != 0 {
			return errors.WithDetailf(errPosition, "invalid position %d for issuance source", vs.Position)
		}
		dest = ref.Witness.Destination

	case *Spend:
		if vs.Position != 0 {
			return errors.WithDetailf(errPosition, "invalid position %d for spend source", vs.Position)
		}
		dest = ref.Witness.Destination

	case *Mux:
		if vs.Position >= uint64(len(ref.Witness.Destinations)) {
			return errors.WithDetailf(errPosition, "invalid position %d for %d-destination mux source", vs.Position, len(ref.Witness.Destinations))
		}
		dest = ref.Witness.Destinations[vs.Position]

	default:
		return errors.WithDetailf(errEntryType, "value source is %T, should be issuance, spend, or mux", vs.Entry)
	}

	if dest.Ref != vstate.entryID {
		return errors.WithDetailf(errMismatchedReference, "value source for %x has disagreeing destination %x", vstate.entryID[:], dest.Ref[:])
	}

	if dest.Position != vstate.sourcePos {
		return fmt.Errorf("value source position %d disagrees with %d", dest.Position, vstate.sourcePos)
	}

	if dest.Value != vs.Value {
		return fmt.Errorf("source value %v disagrees with %v", dest.Value, vs.Value)
	}

	return nil
}

type ValueDestination struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64

	// The Entry corresponding to Ref, if available.
	// ValueDestinations live in the witness half of entries and so
	// aren't hashed; so the entry:"-" struct tag is not needed.
	Entry
}

func (vd *ValueDestination) CheckValid(vs *validationState) error {
	var src ValueSource
	switch ref := vd.Entry.(type) {
	case *Output:
		if vd.Position != 0 {
			return fmt.Errorf("invalid position %d for output destination", vd.Position)
		}
		src = ref.Body.Source

	case *Retirement:
		if vd.Position != 0 {
			return fmt.Errorf("invalid position %d for retirement destination", vd.Position)
		}
		src = ref.Body.Source

	case *Mux:
		if vd.Position >= uint64(len(ref.Body.Sources)) {
			return fmt.Errorf("invalid position %d for %d-source mux destination", vd.Position, len(ref.Body.Sources))
		}
		src = ref.Body.Sources[vd.Position]

	default:
		return fmt.Errorf("value destination is %T, should be output, retirement, or mux", vd.Entry)
	}

	if src.Ref != vs.entryID {
		return fmt.Errorf("value destination for %x has disagreeing source %x", vs.entryID[:], src.Ref[:])
	}

	if src.Position != vs.destPos {
		return fmt.Errorf("value destination position %d disagrees with %d", src.Position, vs.destPos)
	}

	if src.Value != vd.Value {
		return fmt.Errorf("destination value %v disagrees with %v", src.Value, vd.Value)
	}

	return nil
}
