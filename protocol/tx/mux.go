package tx

type mux struct {
	body struct {
		Sources []valueSource
		ExtHash extHash
	}
}

func (mux) Type() string         { return "mux1" }
func (m *mux) Body() interface{} { return m.body }

func (mux) Ordinal() int { return -1 }

func newMux(sources []valueSource) *mux {
	m := new(mux)
	m.body.Sources = sources
	return m
}
