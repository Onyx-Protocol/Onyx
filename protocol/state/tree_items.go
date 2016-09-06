package state

import (
	"bytes"

	"golang.org/x/crypto/sha3"

	"chain/encoding/blockchain"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/patricia"
)

// OutputTreeItem returns the key of an output in the state tree,
// as well as a patricia.Hasher for Inserts into the state tree.
func OutputTreeItem(o *Output) ([]byte, patricia.Hasher) {
	b := bytes.NewBuffer(nil)
	b.Write(o.AssetID[:])
	b.Write([]byte("o"))
	w := errors.NewWriter(b) // used to satisfy interfaces
	o.Outpoint.WriteTo(w)
	return b.Bytes(), outputHasher(*o)
}

type outputHasher Output

func (o outputHasher) Hash() bc.Hash {
	var buf bytes.Buffer
	o.Outpoint.WriteTo(&buf)
	blockchain.WriteUvarint(&buf, o.Amount)
	blockchain.WriteBytes(&buf, o.ControlProgram)
	h := sha3.Sum256(buf.Bytes())
	return h
}
