package bc

import (
	"bytes"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
)

// SigHasher caches a txhash for reuse with multiple inputs.
type SigHasher struct {
	txData *TxData
	txHash *Hash // not computed until needed
}

func NewSigHasher(txData *TxData) *SigHasher {
	return &SigHasher{txData: txData}
}

func (s *SigHasher) Hash(idx uint32) Hash {
	if s.txHash == nil {
		h := s.txData.Hash()
		s.txHash = &h
	}
	h := sha3pool.Get256()
	h.Write((*s.txHash)[:])
	blockchain.WriteVarint31(h, uint64(idx)) // TODO(bobg): check and return error

	var outHash Hash
	inp := s.txData.Inputs[idx]
	si, ok := inp.TypedInput.(*SpendInput)
	if ok {
		// inp is a spend
		var ocBuf bytes.Buffer
		si.OutputCommitment.WriteTo(&ocBuf)
		sha3pool.Sum256(outHash[:], ocBuf.Bytes())
	} else {
		// inp is an issuance
		outHash = EmptyStringHash
	}

	h.Write(outHash[:])
	var hash Hash
	h.Read(hash[:])
	sha3pool.Put256(h)
	return hash
}
