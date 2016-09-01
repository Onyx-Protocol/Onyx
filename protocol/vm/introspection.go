package vm

import (
	"bytes"
	"math"

	"golang.org/x/crypto/sha3"

	"chain/protocol/bc"
)

func opFindOutput(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(16)
	if err != nil {
		return err
	}

	prog, err := vm.pop(true)
	if err != nil {
		return err
	}
	vmVersion, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	assetID, err := vm.pop(true)
	if err != nil {
		return err
	}
	amount, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if amount < 0 {
		return ErrBadValue
	}
	refdatahash, err := vm.pop(true)
	if err != nil {
		return err
	}

	for _, o := range vm.tx.Outputs {
		if o.AssetVersion != 1 {
			continue
		}
		if o.Amount != uint64(amount) {
			continue
		}
		if o.VMVersion != uint32(vmVersion) {
			continue
		}
		if !bytes.Equal(o.ControlProgram, prog) {
			continue
		}
		if !bytes.Equal(o.AssetID[:], assetID) {
			continue
		}
		if len(refdatahash) > 0 {
			h := sha3.Sum256(o.ReferenceData)
			if !bytes.Equal(h[:], refdatahash) {
				continue
			}
		}
		return vm.pushBool(true, true)
	}
	return vm.pushBool(false, true)
}

func opAsset(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	assetID := vm.tx.Inputs[vm.inputIndex].AssetID()
	return vm.push(assetID[:], true)
}

func opAmount(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	amount := vm.tx.Inputs[vm.inputIndex].Amount()
	return vm.pushInt64(int64(amount), true)
}

func opProgram(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	var prog []byte
	inp := vm.tx.Inputs[vm.inputIndex]
	switch c := inp.InputCommitment.(type) {
	case *bc.IssuanceInputCommitment:
		prog = c.IssuanceProgram
	case *bc.SpendInputCommitment:
		prog = c.ControlProgram
	}

	return vm.push(prog, true)
}

func opMinTime(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	return vm.pushInt64(int64(vm.tx.MinTime), true)
}

func opMaxTime(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	maxTime := vm.tx.MaxTime
	if maxTime == 0 || maxTime > math.MaxInt64 {
		maxTime = uint64(math.MaxInt64)
	}

	return vm.pushInt64(int64(maxTime), true)
}

func opRefDataHash(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	h := sha3.Sum256(vm.tx.Inputs[vm.inputIndex].ReferenceData)
	return vm.push(h[:], true)
}

func opIndex(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	return vm.pushInt64(int64(vm.inputIndex), true)
}

func opOutpoint(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	txin := vm.tx.Inputs[vm.inputIndex]
	if txin.IsIssuance() {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	outpoint := txin.Outpoint()

	err = vm.push(outpoint.Hash[:], true)
	if err != nil {
		return err
	}
	return vm.pushInt64(int64(outpoint.Index), true)
}
