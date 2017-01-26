package tx

import (
	"chain/crypto/sha3pool"
	"chain/protocol/bc"
)

type data struct {
	body bc.Hash
}

func (data) Type() string         { return "data1" }
func (d *data) Body() interface{} { return d.body }

func newData(hash bc.Hash) entry {
	d := new(data)
	d.body = hash
	return d
}

func hashData(data []byte) (h bc.Hash) {
	// TODO(kr): domain separation here? spec might need updating
	sha3pool.Sum256(h[:], data)
	return
}
