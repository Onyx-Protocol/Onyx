package tx

type output struct {
	Source         valueSource
	ControlProgram program
	Data           entryRef
	ExtHash        extHash
}

func (output) Type() string { return "output1" }

func newOutput(source valueSource, controlProgram program, data entryRef) *entry {
	return &entry{
		body: &output{
			Source:         source,
			ControlProgram: controlProgram,
			Data:           data,
		},
	}
}
