package tx

import "chain/protocol/bc"

type nonce struct {
	body struct {
		Program   program
		TimeRange bc.Hash
		ExtHash   bc.Hash
	}
}

func (nonce) Type() string         { return "nonce1" }
func (n *nonce) Body() interface{} { return n.body }

func (nonce) Ordinal() int { return -1 }

func newNonce(p program, tr bc.Hash) *nonce {
	n := new(nonce)
	n.body.Program = p
	n.body.TimeRange = tr
	return n
}
