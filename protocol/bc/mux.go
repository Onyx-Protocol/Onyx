package bc

// Mux splits and combines value from one or more source entries,
// making it available to one or more destination entries. It
// satisfies the Entry interface.
type Mux struct {
	body struct {
		Sources []valueSource // issuances, spends, and muxes
		Program Program
		ExtHash Hash
	}

	witness struct {
		Destinations []ValueDestination // outputs, retirements, and muxes
	}
}

func (Mux) Type() string         { return "mux1" }
func (m *Mux) Body() interface{} { return m.body }

func (Mux) Ordinal() int { return -1 }

func (m *Mux) Destinations() []ValueDestination {
	return m.witness.Destinations
}

// NewMux creates a new Mux.
func NewMux(sources []valueSource, program Program) *Mux {
	m := new(Mux)
	m.body.Sources = sources
	m.body.Program = program
	return m
}

func (mux *Mux) CheckValid(state *validationState) error {
	// xxx execute mux program

	for i, src := range mux.body.Sources {
		srcState := *state
		srcState.sourcePosition = i
		err := src.Entry.CheckValid(srcState)
		if err != nil {
			// xxx
		}
	}

	for i, dest := range mux.witness.Destinations {
		destState := *state
		destState.destPosition = i
		err := dest.Entry.CheckValid(destState)
		if err != nil {
			// xxx
		}
	}

	parity := make(map[AssetID]uint64)
	for _, src := range mux.body.Sources {
		sum, ok := checked.AddInt64(parity[src.Value.AssetID], int64(src.Value.Amount))
		if !ok {
			// xxx error
		}
		parity[src.Value.AssetID] = sum
	}

	for _, dest := range mux.witness.Destinations {
		sum, ok := parity[dest.Value.AssetID]
		if !ok {
			// xxx error (dest assetid with no matching src)
		}

		diff, ok := checked.SubInt64(sum, int64(dest.Value.Amount))
		if !ok {
			// xxx error
		}
		parity[dest.Value.AssetID] = diff
	}

	for assetID, amount := range parity {
		if amount != 0 {
			// xxx error
		}
	}

	if state.txVersion == 1 && (mux.body.ExtHash != Hash{}) {
		// xxx error
	}

	return nil
}
