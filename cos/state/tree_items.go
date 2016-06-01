package state

import (
	"bytes"

	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/crypto/hash256"
	"chain/encoding/blockchain"
	"chain/errors"
)

// ADPTreeItem returns the key of an ADP in the state tree,
// as well as a patricia.Valuer for Inserts into the state tree.
func ADPTreeItem(assetID bc.AssetID, adp bc.Hash) ([]byte, patricia.Valuer) {
	return append(assetID[:], byte('d')), patricia.HashValuer(adp)
}

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

// CirculationTreeItem returns the key for circulation
// of an asset in the state tree, as well as a patricia.Valuer
// for Inserts into the state tree.
func CirculationTreeItem(assetID bc.AssetID, amt uint64) ([]byte, patricia.Valuer) {
	return append(assetID[:], byte('c')), uint64Valuer(amt)
}

// GetCirculation extracts the circulation for the provided asset from
// the state tree.
func GetCirculation(tree *patricia.Tree, assetID bc.AssetID) uint64 {
	k := append(assetID[:], byte('c'))
	n := tree.Lookup(k)
	if n == nil {
		return 0
	}

	r := bytes.NewReader(n.Value().Bytes)
	v, _ := blockchain.ReadUint64(r)
	return v
}

type uint64Valuer uint64

func (v uint64Valuer) Value() patricia.Value {
	var buf bytes.Buffer
	blockchain.WriteUint64(&buf, uint64(v))
	return patricia.Value{Bytes: buf.Bytes()}
}

type outputValuer Output

func (o outputValuer) Value() patricia.Value {
	var buf bytes.Buffer
	o.Outpoint.WriteTo(&buf)
	blockchain.WriteUint64(&buf, o.Amount)
	blockchain.WriteBytes(&buf, o.Script)
	h := hash256.Sum(buf.Bytes())
	return patricia.Value{
		Bytes:  h[:],
		IsHash: true,
	}
}
