package bc

import "io"

// Mux splits and combines value from one or more source entries,
// making it available to one or more destination entries. It
// satisfies the Entry interface.

func (Mux) typ() string { return "mux1" }
func (m *Mux) writeForHash(w io.Writer) {
	mustWriteForHash(w, m.Sources)
	mustWriteForHash(w, m.Program)
	mustWriteForHash(w, m.ExtHash)
}

// NewMux creates a new Mux.
func NewMux(sources []*ValueSource, program *Program) *Mux {
	return &Mux{
		Sources: sources,
		Program: program,
	}
}
