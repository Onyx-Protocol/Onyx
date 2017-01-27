package tx

type anchor struct {
	Program   program
	TimeRange entryRef
	ExtHash   extHash
}

func (anchor) Type() string { return "anchor1" }

func newAnchor(p program, tr entryRef) *entry {
	return &entry{
		body: &anchor{
			Program:   p,
			TimeRange: tr,
		},
	}
}
