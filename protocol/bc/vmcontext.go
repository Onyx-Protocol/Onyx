package bc

import (
	"bytes"

	"chain/protocol/vm"
)

func NewBlockVMContext(block *Block, prog []byte, args [][]byte) *vm.Context {
	blockHash := block.Hash().Bytes()
	return &vm.Context{
		VMVersion: 1,
		Code:      prog,
		Arguments: args,

		BlockHash:            &blockHash,
		BlockTimeMS:          &block.TimestampMS,
		NextConsensusProgram: &block.ConsensusProgram,
	}
}

func NewTxVMContext(tx *Tx, index uint32, prog Program, args [][]byte) *vm.Context {
	var (
		txSigHash        = tx.SigHash(index).Bytes()
		numResults       = uint64(len(tx.Results))
		assetID          = tx.Inputs[index].AssetID()
		assetIDBytes     = assetID[:]
		amount           = tx.Inputs[index].Amount()
		inputRefDataHash = hashData(tx.Inputs[index].ReferenceData).Bytes()
		txRefDataHash    = hashData(tx.ReferenceData).Bytes()
	)

	checkOutput := func(index uint64, refdatahash []byte, amount uint64, assetID []byte, vmVersion uint64, code []byte) (bool, error) {
		if index >= uint64(len(tx.Outputs)) {
			return false, vm.ErrBadValue
		}
		o := tx.Outputs[index]
		if o.AssetVersion != 1 {
			return false, nil
		}
		if o.Amount != uint64(amount) {
			return false, nil
		}
		if o.VMVersion != uint64(vmVersion) {
			return false, nil
		}
		if !bytes.Equal(o.ControlProgram, code) {
			return false, nil
		}
		if !bytes.Equal(o.AssetID[:], assetID) {
			return false, nil
		}
		if len(refdatahash) > 0 {
			h := hashData(o.ReferenceData)
			if !bytes.Equal(h[:], refdatahash) {
				return false, nil
			}
		}
		return true, nil
	}

	result := &vm.Context{
		VMVersion: prog.VMVersion,
		Code:      prog.Code,
		Arguments: args,

		TxVersion: &tx.Version,

		TxSigHash:        &txSigHash,
		NumResults:       &numResults,
		AssetID:          &assetIDBytes,
		Amount:           &amount,
		MinTimeMS:        &tx.MinTime,
		MaxTimeMS:        &tx.MaxTime,
		InputRefDataHash: &inputRefDataHash,
		TxRefDataHash:    &txRefDataHash,
		InputIndex:       &index,
		CheckOutput:      checkOutput,
	}
	switch inp := tx.Inputs[index].TypedInput.(type) {
	case *IssuanceInput:
		result.Nonce = &inp.Nonce
	case *SpendInput:
		spentOutputID := tx.SpentOutputIDs[index][:]
		result.SpentOutputID = &spentOutputID
	}

	return result
}
