package bc

// Output is the result of a transfer of value. The value it contains
// may be accessed by a later Spend entry (if that entry can satisfy
// the Output's ControlProgram). Output satisfies the Entry interface.
//
// (Not to be confused with the deprecated type TxOutput.)

func (Output) typ() string          { return "output1" }
func (o *Output) body() interface{} { return o.Body }

// NewOutput creates a new Output.
func NewOutput(source *ValueSource, controlProgram *Program, data *Hash, ordinal uint64) *Output {
	return &Output{
		Body: &Output_Body{
			Source:         source,
			ControlProgram: controlProgram,
			Data:           data,
		},
		Ordinal: ordinal,
	}
}
