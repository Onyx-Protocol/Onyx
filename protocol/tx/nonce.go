package tx

import "chain/protocol/bc"

type nonce struct {
	body struct {
		Program   program
		TimeRange bc.Hash
		ExtHash   bc.Hash
	}

	// TimeRange contains (a pointer to) the manifested entry
	// corresponding to body.TimeRange.
	TimeRange *timeRange
}

func (nonce) Type() string         { return "nonce1" }
func (n *nonce) Body() interface{} { return n.body }

func (nonce) Ordinal() int { return -1 }

func newNonce(p program, tr *timeRange) *nonce {
	n := new(nonce)
	n.body.Program = p
	if tr != nil {
		n.body.TimeRange = entryID(tr)
		n.TimeRange = tr
	}
	return n
}
