package bc

type Mux struct {
	body struct {
		Sources []valueSource
		Program Program
		ExtHash Hash
	}

	// Sources contains (pointers to) the manifested entries for each
	// body.Sources[i].Ref.
	Sources []Entry // each entry is *issuance, *spend, or *mux
}

func (Mux) Type() string         { return "mux1" }
func (m *Mux) Body() interface{} { return m.body }

func (Mux) Ordinal() int { return -1 }

func NewMux(program Program) *Mux {
	m := new(Mux)
	m.body.Program = program
	return m
}

func (m *Mux) addSource(e Entry, value AssetAmount, position uint64) {
	m.addSourceID(EntryID(e), value, position)
	m.Sources[len(m.Sources)-1] = e
}

func (m *Mux) addSourceID(sourceID Hash, value AssetAmount, position uint64) {
	src := valueSource{
		Ref:      sourceID,
		Value:    value,
		Position: position,
	}
	m.body.Sources = append(m.body.Sources, src)
	m.Sources = append(m.Sources, nil)
}
