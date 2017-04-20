package bc

import "io"

// Nonce contains data used, among other things, for distinguishing
// otherwise-identical issuances (when used as those issuances'
// "anchors"). It satisfies the Entry interface.

func (Nonce) typ() string { return "nonce1" }
func (n *Nonce) writeForHash(w io.Writer) {
	mustWriteForHash(w, n.Program)
	mustWriteForHash(w, n.TimeRangeId)
	mustWriteForHash(w, n.ExtHash)
}

// NewNonce creates a new Nonce.
func NewNonce(p *Program, trID *Hash) *Nonce {
	return &Nonce{
		Program:     p,
		TimeRangeId: trID,
	}
}

func (n *Nonce) SetAnchored(id *Hash) {
	n.WitnessAnchoredId = id
}
