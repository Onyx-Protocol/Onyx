package tx

import "chain/protocol/bc"

// idWrapper contains an entry and its hash. It satisfies the entry
// interface (by delegating entry methods to the wrapped entry). When
// passed to entryID, the contained hash is returned rather than being
// recomputed.
type idWrapper struct {
	entry
	bc.Hash
}

// newIDWrapper wraps the given entry in a new idWrapper object. If
// the entry's id is already known, it can be passed in to avoid
// recomputing it; otherwise it will be computed and cached in the new
// idWrapper. If the given entry is already an idWrapper, it's
// returned as-is with no new wrapper created.
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
