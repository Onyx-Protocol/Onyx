package tx

type retirement struct {
	Source  valueSource
	Data    entryRef
	ExtHash extHash
}

func (retirement) Type() string { return "retirement1" }

func newRetirement(source valueSource, data entryRef) *entry {
	return &entry{
		body: &retirement{
			Source: source,
			Data:   data,
		},
	}
}
