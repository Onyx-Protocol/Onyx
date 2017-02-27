package tx

import "chain/protocol/bc"

type mux struct {
	body struct {
		Sources []valueSource
		Program program
		ExtHash bc.Hash
	}
}

func (mux) Type() string         { return "mux1" }
func (m *mux) Body() interface{} { return m.body }

func (mux) Ordinal() int { return -1 }

func newMux(sources []valueSource, program program) *mux {
	m := new(mux)
	m.body.Sources = sources
	m.body.Program = program
	return m
}
