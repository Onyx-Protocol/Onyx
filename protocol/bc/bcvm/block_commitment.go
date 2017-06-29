package bcvm

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
	ConsensusPubkeys [][]byte
	ConsensusQuorum  uint32
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
	bc.ConsensusPubkeys, err = blockchain.ReadVarstrList(r)
	if err != nil {
		return err
	}
	bc.ConsensusQuorum, err = blockchain.ReadVarint31(r)
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
	_, err = blockchain.WriteVarstrList(w, bc.ConsensusPubkeys)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint31(w, uint64(bc.ConsensusQuorum))
	return err
}
