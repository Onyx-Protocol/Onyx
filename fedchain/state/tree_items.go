package state

import (
	"bytes"

	"chain/crypto/hash256"
	"chain/encoding/blockchain"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/patricia"
)

// ADPTreeItem returns the key of an ADP in the state tree,
// as well as a patricia.Hasher for Inserts into the state tree.
func ADPTreeItem(assetID bc.AssetID, adp bc.Hash) ([]byte, patricia.Hasher) {
	return append(assetID[:], byte('d')), defHasher(adp)
}

type defHasher bc.Hash

func (d defHasher) Hash() bc.Hash {
	return bc.Hash(d)
}

// OutputTreeItem returns the key of an output in the state tree,
// as well as a patricia.Hasher for Inserts into the state tree.
func OutputTreeItem(o *Output) ([]byte, patricia.Hasher) {
	b := bytes.NewBuffer(nil)
	b.Write(o.AssetID[:])
	b.Write([]byte("o"))
	w := errors.NewWriter(b) // used to satisfy interfaces
	o.Outpoint.WriteTo(w)
	return b.Bytes(), (*outputHasher)(o)
}

type outputHasher Output

func (o *outputHasher) Hash() bc.Hash {
	h := hash256.New()
	w := errors.NewWriter(h) // used to satisfy interfaces
	o.Outpoint.WriteTo(w)
	blockchain.WriteUint64(w, o.Amount)
	blockchain.WriteBytes(w, o.Script)

	var bcHash bc.Hash
	h.Sum(bcHash[:0])
	return bcHash
}

// CirculationTreeItem returns the key for circulation
// of an asset in the state tree, as well as a patricia.Hasher
// for Inserts into the state tree.
func CirculationTreeItem(assetID bc.AssetID, amt uint64) ([]byte, patricia.Hasher) {
	return append(assetID[:], byte('c')), numHasher(amt)
}

type numHasher uint64

func (n numHasher) Hash() bc.Hash {
	h := hash256.New()
	w := errors.NewWriter(h)
	blockchain.WriteUint64(w, uint64(n))

	var bcHash bc.Hash
	h.Sum(bcHash[:0])
	return bcHash
}
