package bc

// Nonce contains data used, among other things, for distinguishing
// otherwise-identical issuances (when used as those issuances'
// "anchors"). It satisfies the Entry interface.
type Nonce struct {
	Body struct {
		Program   Program
		TimeRange Hash
		ExtHash   Hash
	}

	// TimeRange contains (a pointer to) the manifested entry
	// corresponding to Body.TimeRange.
	TimeRange *TimeRange
}

func (Nonce) Type() string         { return "nonce1" }
func (n *Nonce) body() interface{} { return n.Body }

func (Nonce) Ordinal() int { return -1 }

// NewNonce creates a new Nonce.
func NewNonce(p Program, tr *TimeRange) *Nonce {
	n := new(Nonce)
	n.Body.Program = p
	n.Body.TimeRange = EntryID(tr)
	n.TimeRange = tr
	return n
}
