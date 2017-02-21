package tx

import "chain/protocol/bc"

type output struct {
	body struct {
		Source         valueSource
		ControlProgram program
		RefDataHash    bc.Hash
		ExtHash        extHash
	}
	ordinal int
}

func (output) Type() string         { return "output1" }
func (o *output) Body() interface{} { return o.body }

func (o output) Ordinal() int { return o.ordinal }

func newOutput(source valueSource, controlProgram program, refDataHash bc.Hash, ordinal int) *output {
	out := new(output)
	out.body.Source = source
	out.body.ControlProgram = controlProgram
	out.body.RefDataHash = refDataHash
	out.ordinal = ordinal
	return out
}
