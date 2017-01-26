package bc

import (
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
)

// OutputID identifies previous transaction output in transaction inputs.
type OutputID struct{ Hash }

// UnspentID identifies and commits to unspent output.
type UnspentID struct{ Hash }

// ComputeOutputID computes the output ID defined by transaction hash and output index.
func ComputeOutputID(txHash Hash, outputIndex uint32) (oid OutputID) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(txHash[:])
	blockchain.WriteVarint31(h, uint64(outputIndex))
	h.Read(oid.Hash[:])
	return oid
}

// ComputeOutputID computes the output ID defined by transaction hash, output index and output hash.
func ComputeUnspentID(oid OutputID, outputHash Hash) (uid UnspentID) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(oid.Hash[:])
	h.Write(outputHash[:])
	h.Read(uid.Hash[:])
	return uid
}

// WriteTo writes p to w.
func (outid *OutputID) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(outid.Hash[:])
	return int64(n), err
}

func (outid *OutputID) readFrom(r io.Reader) (int, error) {
	return io.ReadFull(r, outid.Hash[:])
}
