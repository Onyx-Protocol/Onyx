package bc

import (
	"io"

	"chain/encoding/blockchain"
)

type BlockCommitment struct {
	// TransactionsMerkleRoot is the root hash of the Merkle binary hash
	// tree formed by the transaction witness hashes of all transactions
	// included in the block.
	TransactionsMerkleRoot Hash

	// AssetsMerkleRoot is the root hash of the Merkle Patricia Tree of
	// the set of unspent outputs with asset version 1 after applying
	// the block.
	AssetsMerkleRoot Hash

	// ConsensusProgram is the predicate for validating the next block.
	ConsensusProgram []byte
}

func (bc *BlockCommitment) readFrom(r io.Reader) error {
	_, err := io.ReadFull(r, bc.TransactionsMerkleRoot[:])
	if err != nil {
		return err
	}
	_, err = io.ReadFull(r, bc.AssetsMerkleRoot[:])
	if err != nil {
		return err
	}
	bc.ConsensusProgram, _, err = blockchain.ReadVarstr31(r)
	return err
}

func (bc *BlockCommitment) writeTo(w io.Writer) error {
	_, err := w.Write(bc.TransactionsMerkleRoot[:])
	if err != nil {
		return err
	}
	_, err = w.Write(bc.AssetsMerkleRoot[:])
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstr31(w, bc.ConsensusProgram)
	return err
}
