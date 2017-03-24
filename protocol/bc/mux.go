package bc

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
