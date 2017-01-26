package tx

type anchor struct {
	body struct {
		Program   program
		TimeRange entryRef
		ExtHash   extHash
	}
}

func (anchor) Type() string         { return "anchor1" }
func (a *anchor) Body() interface{} { return a.body }

func newAnchor(p program, tr entryRef) *anchor {
	a := new(anchor)
	a.body.Program = p
	a.body.TimeRange = tr
	return a
}
