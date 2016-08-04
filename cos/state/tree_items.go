package state

import (
	"bytes"

	"golang.org/x/crypto/sha3"

	"chain/cos/patricia"
	"chain/encoding/blockchain"
	"chain/errors"
)

// OutputTreeItem returns the key of an output in the state tree,
// as well as a patricia.Valuer for Inserts into the state tree.
func OutputTreeItem(o *Output) ([]byte, patricia.Valuer) {
	b := bytes.NewBuffer(nil)
	b.Write(o.AssetID[:])
	b.Write([]byte("o"))
	w := errors.NewWriter(b) // used to satisfy interfaces
	o.Outpoint.WriteTo(w)
	return b.Bytes(), outputValuer(*o)
}

type outputValuer Output

func (o outputValuer) Value() patricia.Value {
	var buf bytes.Buffer
	o.Outpoint.WriteTo(&buf)
	blockchain.WriteUvarint(&buf, o.Amount)
	blockchain.WriteBytes(&buf, o.ControlProgram)
	h := sha3.Sum256(buf.Bytes())
	return patricia.Value{
		Bytes:  h[:],
		IsHash: true,
	}
}
