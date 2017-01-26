package tx

type anchor struct {
	body struct {
		program   program
		timeRange entryRef
		extHash   extHash
	}
}

func (anchor) Type() string         { return "anchor1" }
func (a *anchor) Body() interface{} { return a.body }

func newAnchor(p program, tr entryRef) *anchor {
	a := new(anchor)
	a.body.program = p
	a.body.timeRange = tr
	return a
}
