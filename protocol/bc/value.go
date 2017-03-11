package bc

import (
	"fmt"

	"chain/errors"
)

type valueSource struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64

	// The Entry corresponding to Ref, if available
	// The struct tag excludes the field from hashing
	Entry `entry:"-"`
}

// CheckValid checks the validity of a value source in the context of
// its containing entry.
func (vs *valueSource) CheckValid(entry Entry, srcPos uint64) error {
	// xxx check that Entry's ID equals Ref?

	err := vs.Entry.CheckValid()
	if err != nil {
		return errors.Wrap(err, "checking value source")
	}

	var dest ValueDestination
	switch ref := vs.Entry.(type) {
	case *Issuance:
		if vs.Position != 0 {
			return fmt.Errorf("invalid position %d for issuance source", vs.Position)
		}
		dest = ref.witness.Destination

	case *Spend:
		if vs.Position != 0 {
			return fmt.Errorf("invalid position %d for spend source", vs.Position)
		}
		dest = ref.witness.Destination

	case *Mux:
		if vs.Position >= len(ref.witness.Destinations) {
			return fmt.Errorf("invalid position %d for %d-destination mux source", vs.Position, len(ref.witness.Destinations))
		}
		dest = ref.witness.Destinations[vs.Position]

	default:
		return fmt.Errorf("value source is %T, should be issuance, spend, or mux", vs.Entry)
	}

	id := EntryID(entry)
	if dest.Ref != id {
		return fmt.Errorf("value source for %x has disagreeing destination %x", id[:], dest.Ref[:])
	}

	if dest.Position != srcPos {
		return fmt.Errorf("value source position %d disagrees with %d", dest.Position, srcPos)
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

	// The Entry corresponding to Ref, if available
	// The struct tag excludes the field from hashing
	Entry `entry:"-"`
}

func (vd *ValueDestination) CheckValid(entry Entry, destPos uint64) error {
	// xxx check reachability of vd.Ref from transaction

	var src valueSource
	switch ref := vd.Entry.(type) {
	case *Output:
		if vd.Position != 0 {
			fmt.Errorf("invalid position %d for output destination", vd.Position)
		}
		src = ref.body.Source

	case *Retirement:
		if vd.Position != 0 {
			fmt.Errorf("invalid position %d for retirement destination", vd.Position)
		}
		src = ref.body.Source

	case *Mux:
		if vd.Position >= len(ref.body.Sources) {
			return fmt.Errorf("invalid position %d for %d-source mux destination", vd.Position, len(ref.body.Sources))
		}
		src = ref.body.Sources[vd.Position]

	default:
		return fmt.Errorf("value destination is %T, should be output, retirement, or mux", vd.Entry)
	}

	id := EntryID(entry)
	if src.Ref != id {
		return fmt.Errorf("value destination for %x has disagreeing source %x", id[:]. src.Ref[:])
	}

	if src.Position != destPos {
		return fmt.Errorf("value destination position %d disagrees with %d", src.Position, destPos)
	}

	if src.Value != vd.Value {
		return fmt.Errorf("destination value %v disagrees with %v", src.Value, vd.Value)
	}

	return nil
}
