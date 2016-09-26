package state

import (
	"bytes"

	"golang.org/x/crypto/sha3"

	"chain/encoding/blockchain"
	"chain/errors"
	"chain/protocol/bc"
)

// OutputTreeItem returns the key of an output in the state tree,
// as well as a bc.Hash for Inserts into the state tree.
func OutputTreeItem(o *Output) ([]byte, bc.Hash) {
	b := bytes.NewBuffer(nil)
	b.Write(o.AssetID[:])
	b.Write([]byte("o"))
	w := errors.NewWriter(b) // used to satisfy interfaces
	o.Outpoint.WriteTo(w)
	return b.Bytes(), hashOutput(*o)
}

func hashOutput(o Output) bc.Hash {
	var buf bytes.Buffer
	o.Outpoint.WriteTo(&buf)
	blockchain.WriteUvarint(&buf, o.Amount)
	blockchain.WriteBytes(&buf, o.ControlProgram)
	h := sha3.Sum256(buf.Bytes())
	return h
}
