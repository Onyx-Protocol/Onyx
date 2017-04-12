package bc

// Nonce contains data used, among other things, for distinguishing
// otherwise-identical issuances (when used as those issuances'
// "anchors"). It satisfies the Entry interface.

func (Nonce) Type() string         { return "nonce1" }
func (n *Nonce) body() interface{} { return n.Body }

// NewNonce creates a new Nonce.
func NewNonce(p *Program, trID *Hash) *Nonce {
	return &Nonce{
		Body: &Nonce_Body{
			Program:     p,
			TimeRangeId: trID,
		},
		Witness: &Nonce_Witness{},
	}
}

func (n *Nonce) SetAnchored(id *Hash) {
	n.Witness.AnchoredId = id
}
