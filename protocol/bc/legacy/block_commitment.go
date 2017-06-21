package legacy

import (
	"io"

	"chain/encoding/blockchain"
	"chain/protocol/bc"
)

type BlockCommitment struct {
	// TransactionsMerkleRoot is the root hash of the Merkle binary hash
	// tree formed by the hashes of all transactions included in the
	// block.
	TransactionsMerkleRoot bc.Hash

	// AssetsMerkleRoot is the root hash of the Merkle Patricia Tree of
	// the set of unspent outputs with asset version 1 after applying
	// the block.
	AssetsMerkleRoot bc.Hash

	// ConsensusProgram is the predicate for validating the next block.
	ConsensusProgram []byte
}

func (bc *BlockCommitment) readFrom(r blockchain.Reader) error {
	_, err := bc.TransactionsMerkleRoot.ReadFrom(r)
	if err != nil {
		return err
	}
	_, err = bc.AssetsMerkleRoot.ReadFrom(r)
	if err != nil {
		return err
	}
	bc.ConsensusProgram, err = blockchain.ReadVarstr31(r)
	return err
}

func (bc *BlockCommitment) writeTo(w io.Writer) error {
	_, err := bc.TransactionsMerkleRoot.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = bc.AssetsMerkleRoot.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstr31(w, bc.ConsensusProgram)
	return err
}
