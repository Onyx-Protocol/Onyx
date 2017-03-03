package tx

import "chain/protocol/bc"

type output struct {
	body struct {
		Source         valueSource
		ControlProgram program
		Data           bc.Hash
		ExtHash        bc.Hash
	}
	ordinal int

	// Source contains (a pointer to) the manifested entry corresponding
	// to body.Source.
	Source entry // *issuance, *spend, or *mux
}

func (output) Type() string         { return "output1" }
func (o *output) Body() interface{} { return o.body }

func (o output) Ordinal() int { return o.ordinal }

func newOutput(controlProgram program, data bc.Hash, ordinal int) *output {
	out := new(output)
	out.body.ControlProgram = controlProgram
	out.body.Data = data
	out.ordinal = ordinal
	return out
}

// setSource is for when you have a complete source entry (e.g. a
// *mux) for an output. When you don't (you only have the ID of the
// source), use setSourceID, below.
func (o *output) setSource(e entry, value bc.AssetAmount, position uint64) {
	w := newIDWrapper(e, nil)
	o.setSourceID(w.Hash, value, position)
	o.Source = w
}

func (o *output) setSourceID(sourceID bc.Hash, value bc.AssetAmount, position uint64) {
	o.body.Source = valueSource{
		Ref:      sourceID,
		Value:    value,
		Position: position,
	}
	o.Source = nil
}
