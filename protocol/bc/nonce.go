package bc

// Nonce contains data used, among other things, for distinguishing
// otherwise-identical issuances (when used as those issuances'
// "anchors"). It satisfies the Entry interface.
type Nonce struct {
	body struct {
		Program   Program
		TimeRange Hash
		ExtHash   Hash
	}

	witness struct {
		Arguments [][]byte
		Anchored Entry
	}

	// TimeRange contains (a pointer to) the manifested entry
	// corresponding to body.TimeRange.
	TimeRange *TimeRange

	// Anchored contains a pointer to the manifested entry corresponding
	// to witness.Anchored.
	Anchored Entry
}

func (Nonce) Type() string         { return "nonce1" }
func (n *Nonce) Body() interface{} { return n.body }

func (Nonce) Ordinal() int { return -1 }

// NewNonce creates a new Nonce.
func NewNonce(p Program, tr *TimeRange) *Nonce {
	n := new(Nonce)
	n.body.Program = p
	n.body.TimeRange = EntryID(tr)
	n.TimeRange = tr
	return n
}

func (n *Nonce) CheckValid(state *validationState) error {
	// xxx eval program

	// xxx
}
