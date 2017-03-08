package bc

type Nonce struct {
	body struct {
		Program   Program
		TimeRange Hash
		ExtHash   Hash
	}

	// TimeRange contains (a pointer to) the manifested entry
	// corresponding to body.TimeRange.
	TimeRange *TimeRange
}

func (Nonce) Type() string         { return "nonce1" }
func (n *Nonce) Body() interface{} { return n.body }

func (Nonce) Ordinal() int { return -1 }

func NewNonce(p Program, tr *TimeRange) *Nonce {
	n := new(Nonce)
	n.body.Program = p
	n.body.TimeRange = EntryID(tr)
	n.TimeRange = tr
	return n
}
