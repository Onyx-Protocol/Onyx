package bc

// Mux splits and combines value from one or more source entries,
// making it available to one or more destination entries. It
// satisfies the Entry interface.

func (Mux) typ() string          { return "mux1" }
func (m *Mux) body() interface{} { return m.Body }

// NewMux creates a new Mux.
func NewMux(sources []*ValueSource, program *Program) *Mux {
	return &Mux{
		Body: &Mux_Body{
			Sources: sources,
			Program: program,
		},
		Witness: &Mux_Witness{},
	}
}
