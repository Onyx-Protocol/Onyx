package tx

type spend struct {
	SpentOutput entryRef // must be an Output entry
	Reference   entryRef // must be a Data entry
	ExtHash     extHash
}

func (spend) Type() string { return "spend1" }

func newSpend(spentOutput, reference entryRef) *entry {
	return &entry{
		body: &spend{
			SpentOutput: spentOutput,
			Reference:   reference,
		},
	}
}
