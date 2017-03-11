package bc

// Output is the result of a transfer of value. The value it contains
// may be accessed by a later Spend entry (if that entry can satisfy
// the Output's ControlProgram). Output satisfies the Entry interface.
//
// (Not to be confused with the deprecated type TxOutput.)
type Output struct {
	body struct {
		Source         valueSource
		ControlProgram Program
		Data           Hash
		ExtHash        Hash
	}
	ordinal int

	// Source contains (a pointer to) the manifested entry corresponding
	// to body.Source.
	Source Entry // *issuance, *spend, or *mux
}

func (Output) Type() string         { return "output1" }
func (o *Output) Body() interface{} { return o.body }

func (o Output) Ordinal() int { return o.ordinal }

// NewOutput creates a new Output. Once created, its source should be
// set with setSource or setSourceID.
func NewOutput(controlProgram Program, data Hash, ordinal int) *Output {
	out := new(Output)
	out.body.ControlProgram = controlProgram
	out.body.Data = data
	out.ordinal = ordinal
	return out
}

// setSource is for when you have a complete source entry (e.g. a
// *mux) for an output. When you don't (you only have the ID of the
// source), use setSourceID, below.
func (o *Output) setSource(e Entry, value AssetAmount, position uint64) {
	o.setSourceID(EntryID(e), value, position)
	o.Source = e
}

func (o *Output) setSourceID(sourceID Hash, value AssetAmount, position uint64) {
	o.body.Source = valueSource{
		Ref:      sourceID,
		Value:    value,
		Position: position,
	}
	o.Source = nil
}
