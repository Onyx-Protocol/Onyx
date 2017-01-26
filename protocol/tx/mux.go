package tx

type mux struct {
	body struct {
		sources []valueSource
	}
}

func (mux) Type() string         { return "mux1" }
func (m *mux) Body() interface{} { return m.body }

func newMux(sources []valueSource) *mux {
	m := new(mux)
	m.body.sources = sources
	return m
}
