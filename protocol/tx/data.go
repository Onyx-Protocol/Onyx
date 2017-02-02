package tx

import (
	"chain/crypto/sha3pool"
	"chain/protocol/bc"
)

// A "Data" entry represents some arbitrary data
// the transaction author wants the current transaction to commit to,
// either for use in programs in the current or future transactions,
// or for reference by external systems.
// This is done with a hash commitment:
// the entry itself stores a 32-byte hash of the underlying data,
// which may be of any length.
// It is the responsibility of the transport layer
// to provide the underlying data
// alongside the actual transaction, if necessary.
// The data need not be made available to all parties;
// it is fine to keep it confidential.
//
// Note that the body of this entry is a hash (of the underlying data);
// the body_hash is a hash of that hash.
type data struct {
	body bc.Hash
}

func (data) Type() string         { return "data1" }
func (d *data) Body() interface{} { return d.body }

func (data) Ordinal() int { return -1 }

func newData(hash bc.Hash) entry {
	d := new(data)
	d.body = hash
	return d
}

func hashData(data []byte) (h bc.Hash) {
	// TODO: Do we want domain separation here?  (E.g., a "data:"
	// prefix.) If so, both the code and the spec need updating.
	sha3pool.Sum256(h[:], data)
	return
}
