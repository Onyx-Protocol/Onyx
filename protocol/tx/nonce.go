package tx

type nonce struct {
	body struct {
		Program   program
		TimeRange entryRef
		ExtHash   extHash
	}
}

func (nonce) Type() string         { return "nonce1" }
func (n *nonce) Body() interface{} { return n.body }

func (nonce) Ordinal() int { return -1 }

func newNonce(p program, tr entryRef) *nonce {
	n := new(nonce)
	n.body.Program = p
	n.body.TimeRange = tr
	return n
}
