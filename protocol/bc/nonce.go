package bc

// Nonce contains data used, among other things, for distinguishing
// otherwise-identical issuances (when used as those issuances'
// "anchors"). It satisfies the Entry interface.

func (Nonce) Type() string         { return "nonce1" }
func (n *Nonce) body() interface{} { return n.Body }

// NewNonce creates a new Nonce.
func NewNonce(p *Program, tr *TimeRange) *Nonce {
	n := new(Nonce)
	n.Body.Program = p
	n.Body.TimeRangeId = EntryID(tr).Proto()
	return n
}

func (n *Nonce) SetAnchored(id Hash) {
	n.Witness.AnchoredId = id.Proto()
}
