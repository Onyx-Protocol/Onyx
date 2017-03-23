package bc

// Mux splits and combines value from one or more source entries,
// making it available to one or more destination entries. It
// satisfies the Entry interface.
type Mux struct {
	Body struct {
		Sources []ValueSource
		Program Program
		ExtHash Hash
	}

	// Sources contains (pointers to) the manifested entries for each
	// Body.Sources[i].Ref.
	Sources []Entry // each entry is *issuance, *spend, or *mux
}

func (Mux) Type() string         { return "mux1" }
func (m *Mux) body() interface{} { return m.Body }

func (Mux) Ordinal() int { return -1 }

// NewMux creates a new Mux. Once created, its sources should be added
// with addSource or addSourceID.
func NewMux(program Program) *Mux {
	m := new(Mux)
	m.Body.Program = program
	return m
}

func (m *Mux) addSource(e Entry, value AssetAmount, position uint64) {
	m.addSourceID(EntryID(e), value, position)
	m.Sources[len(m.Sources)-1] = e
}

func (m *Mux) addSourceID(sourceID Hash, value AssetAmount, position uint64) {
	src := ValueSource{
		Ref:      sourceID,
		Value:    value,
		Position: position,
	}
	m.Body.Sources = append(m.Body.Sources, src)
	m.Sources = append(m.Sources, nil)
}
