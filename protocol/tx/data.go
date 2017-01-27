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
type data bc.Hash

func (data) Type() string { return "data1" }

func newData(hash bc.Hash) *entry {
	return &entry{
		body: (*data)(&hash), // xxx data(hash) would be simpler but is it better to be consistent about entry.body always being a pointer?
	}
}

func hashData(data []byte) (h bc.Hash) {
	// xxx need domain separation here. spec might need updating
	sha3pool.Sum256(h[:], data)
	return
}
