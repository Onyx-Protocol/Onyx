package bc

import (
	"io"

	"chain/encoding/blockchain"
)

type BlockWitness struct {
	// Witness is a vector of arguments to the previous block's
	// ConsensusProgram for validating this block.
	Witness [][]byte
}

func (bw *BlockWitness) readFrom(r io.Reader) (err error) {
	bw.Witness, _, err = blockchain.ReadVarstrList(r)
	return err
}

func (bw *BlockWitness) writeTo(w io.Writer) error {
	_, err := blockchain.WriteVarstrList(w, bw.Witness)
	return err
}
