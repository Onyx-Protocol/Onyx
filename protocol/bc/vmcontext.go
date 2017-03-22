package bc

import (
	"bytes"

	"chain/protocol/vm"
)

// BlockVMContext satisfies the vm.Context interface
type BlockVMContext struct {
	prog  Program
	args  [][]byte
	block *Block
}

func (b *BlockVMContext) VMVersion() uint64   { return b.prog.VMVersion }
func (b *BlockVMContext) Code() []byte        { return b.prog.Code }
func (b *BlockVMContext) Arguments() [][]byte { return b.args }

func (b *BlockVMContext) BlockHash() ([]byte, error) {
	h := b.block.Hash()
	return h[:], nil
}
func (b *BlockVMContext) BlockTimeMS() (uint64, error) { return b.block.TimestampMS, nil }

func (b *BlockVMContext) NextConsensusProgram() ([]byte, error) {
	return b.block.ConsensusProgram, nil
}

func (b *BlockVMContext) TxVersion() (uint64, bool)         { return 0, false }
func (b *BlockVMContext) TxSigHash() ([]byte, error)        { return nil, vm.ErrContext }
func (b *BlockVMContext) NumResults() (uint64, error)       { return 0, vm.ErrContext }
func (b *BlockVMContext) AssetID() ([]byte, error)          { return nil, vm.ErrContext }
func (b *BlockVMContext) Amount() (uint64, error)           { return 0, vm.ErrContext }
func (b *BlockVMContext) MinTimeMS() (uint64, error)        { return 0, vm.ErrContext }
func (b *BlockVMContext) MaxTimeMS() (uint64, error)        { return 0, vm.ErrContext }
func (b *BlockVMContext) InputRefDataHash() ([]byte, error) { return nil, vm.ErrContext }
func (b *BlockVMContext) TxRefDataHash() ([]byte, error)    { return nil, vm.ErrContext }
func (b *BlockVMContext) InputIndex() (uint64, error)       { return 0, vm.ErrContext }
func (b *BlockVMContext) Nonce() ([]byte, error)            { return nil, vm.ErrContext }
func (b *BlockVMContext) SpentOutputID() ([]byte, error)    { return nil, vm.ErrContext }

func (b *BlockVMContext) CheckOutput(uint64, []byte, uint64, []byte, uint64, []byte) (bool, error) {
	return false, vm.ErrContext
}

func NewBlockVMContext(block *Block, prog []byte, args [][]byte) *BlockVMContext {
	return &BlockVMContext{
		prog: Program{
			VMVersion: 1,
			Code:      prog,
		},
		args:  args,
		block: block,
	}
}

type TxVMContext struct {
	prog  Program
	args  [][]byte
	tx    *Tx
	index uint32
}

func NewTxVMContext(tx *Tx, index uint32, prog Program, args [][]byte) *TxVMContext {
	return &TxVMContext{
		prog:  prog,
		args:  args,
		tx:    tx,
		index: index,
	}
}

func (t *TxVMContext) VMVersion() uint64   { return t.prog.VMVersion }
func (t *TxVMContext) Code() []byte        { return t.prog.Code }
func (t *TxVMContext) Arguments() [][]byte { return t.args }

func (t *TxVMContext) BlockHash() ([]byte, error)   { return nil, vm.ErrContext }
func (t *TxVMContext) BlockTimeMS() (uint64, error) { return 0, vm.ErrContext }

func (t *TxVMContext) NextConsensusProgram() ([]byte, error) { return nil, vm.ErrContext }

func (t *TxVMContext) TxVersion() (uint64, bool) { return t.tx.Version, true }

func (t *TxVMContext) TxSigHash() ([]byte, error) {
	h := t.tx.SigHash(t.index)
	return h[:], nil
}

func (t *TxVMContext) NumResults() (uint64, error) { return uint64(len(t.tx.Results)), nil }

func (t *TxVMContext) AssetID() ([]byte, error) {
	a := t.tx.Inputs[t.index].AssetID()
	return a[:], nil
}

func (t *TxVMContext) Amount() (uint64, error) {
	return t.tx.Inputs[t.index].Amount(), nil
}

func (t *TxVMContext) MinTimeMS() (uint64, error) { return t.tx.MinTime, nil }
func (t *TxVMContext) MaxTimeMS() (uint64, error) { return t.tx.MaxTime, nil }

func (t *TxVMContext) InputRefDataHash() ([]byte, error) {
	h := hashData(t.tx.Inputs[t.index].ReferenceData)
	return h[:], nil
}

func (t *TxVMContext) TxRefDataHash() ([]byte, error) {
	h := hashData(t.tx.ReferenceData)
	return h[:], nil
}

func (t *TxVMContext) InputIndex() (uint64, error) {
	return uint64(t.index), nil
}

func (t *TxVMContext) Nonce() ([]byte, error) {
	if inp, ok := t.tx.Inputs[t.index].TypedInput.(*IssuanceInput); ok {
		return inp.Nonce, nil
	}
	return nil, vm.ErrContext
}

func (t *TxVMContext) SpentOutputID() ([]byte, error) {
	if _, ok := t.tx.Inputs[t.index].TypedInput.(*SpendInput); ok {
		return t.tx.TxHashes.SpentOutputIDs[t.index][:], nil
	}
	return nil, vm.ErrContext
}

func (t *TxVMContext) CheckOutput(index uint64, refdatahash []byte, amount uint64, assetID []byte, vmVersion uint64, code []byte) (bool, error) {
	if index >= uint64(len(t.tx.Outputs)) {
		return false, vm.ErrBadValue
	}

	o := t.tx.Outputs[index]
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
