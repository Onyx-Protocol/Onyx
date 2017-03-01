package tx

import "chain/protocol/bc"

type idWrapper struct {
	entry
	bc.Hash
}

func newIDWrapper(e entry, id *bc.Hash) *idWrapper {
	if e, ok := e.(*idWrapper); ok {
		// e is already an idWrapper, don't create a new one.
		return e
	}
	res := &idWrapper{entry: e}
	if id == nil {
		eid := entryID(e)
		id = &eid
	}
	res.Hash = *id
	return res
}

func (w *idWrapper) Type() string {
	return w.entry.Type()
}

func (w *idWrapper) Body() interface{} {
	return w.entry.Body()
}

func (w *idWrapper) Ordinal() int {
	return w.entry.Ordinal()
}
