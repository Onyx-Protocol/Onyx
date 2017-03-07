package tx

import "chain/protocol/bc"

type mux struct {
	body struct {
		Sources []valueSource
		Program program
		ExtHash bc.Hash
	}

	// Sources contains (pointers to) the manifested entries for each
	// body.Sources[i].Ref.
	Sources []entry // each entry is *issuance, *spend, or *mux
}

func (mux) Type() string         { return "mux1" }
func (m *mux) Body() interface{} { return m.body }

func (mux) Ordinal() int { return -1 }

func newMux(program program) *mux {
	m := new(mux)
	m.body.Program = program
	return m
}

func (m *mux) addSource(e entry, value bc.AssetAmount, position uint64) {
	m.addSourceID(entryID(e), value, position)
	m.Sources[len(m.Sources)-1] = e
}

func (m *mux) addSourceID(sourceID bc.Hash, value bc.AssetAmount, position uint64) {
	src := valueSource{
		Ref:      sourceID,
		Value:    value,
		Position: position,
	}
	m.body.Sources = append(m.body.Sources, src)
	m.Sources = append(m.Sources, nil)
}
