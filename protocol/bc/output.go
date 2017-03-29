package bc

// Output is the result of a transfer of value. The value it contains
// may be accessed by a later Spend entry (if that entry can satisfy
// the Output's ControlProgram). Output satisfies the Entry interface.
//
// (Not to be confused with the deprecated type TxOutput.)
type Output struct {
	Body struct {
		Source         ValueSource
		ControlProgram Program
		Data           Hash
		ExtHash        Hash
	}
	ordinal int
}

func (Output) Type() string         { return "output1" }
func (o *Output) body() interface{} { return o.Body }

func (o Output) Ordinal() int { return o.ordinal }

// NewOutput creates a new Output.
func NewOutput(source ValueSource, controlProgram Program, data Hash, ordinal int) *Output {
	out := new(Output)
	out.Body.Source = source
	out.Body.ControlProgram = controlProgram
	out.Body.Data = data
	out.ordinal = ordinal
	return out
}
