package tx

type mux struct {
	Sources []valueSource
}

func (mux) Type() string { return "mux1" }

func newMux(sources []valueSource) *entry {
	return &entry{
		body: &mux{
			Sources: sources,
		},
	}
}
