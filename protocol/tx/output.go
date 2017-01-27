package tx

type output struct {
	Source         valueSource
	ControlProgram program
	Reference      entryRef
	ExtHash        extHash
}

func (output) Type() string { return "output1" }

func newOutput(source valueSource, controlProgram program, reference entryRef) *entry {
	return &entry{
		body: &output{
			Source:         source,
			ControlProgram: controlProgram,
			Reference:      reference,
		},
	}
}
