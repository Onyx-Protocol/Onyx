package vm

import (
	"bytes"
	"fmt"
	"math"

	"golang.org/x/crypto/sha3"

	"chain/protocol/bc"
)

func opCheckOutput(vm *virtualMachine) error {
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
	if vmVersion < 0 {
		return ErrBadValue
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
	index, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if index < 0 || int64(len(vm.tx.Outputs)) <= index {
		return ErrBadValue
	}

	o := vm.tx.Outputs[index]

	if o.AssetVersion != 1 {
		return vm.pushBool(false, true)
	}
	if o.Amount != uint64(amount) {
		return vm.pushBool(false, true)
	}
	if o.VMVersion != uint64(vmVersion) {
		return vm.pushBool(false, true)
	}
	if !bytes.Equal(o.ControlProgram, prog) {
		return vm.pushBool(false, true)
	}
	if !bytes.Equal(o.AssetID[:], assetID) {
		return vm.pushBool(false, true)
	}
	if len(refdatahash) > 0 {
		h := sha3.Sum256(o.ReferenceData)
		if !bytes.Equal(h[:], refdatahash) {
			return vm.pushBool(false, true)
		}
	}
	return vm.pushBool(true, true)
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

	return vm.push(vm.tx.Inputs[vm.inputIndex].Program(), true)
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

func opTxRefDataHash(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	h := sha3.Sum256(vm.tx.ReferenceData)
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

func opNonce(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	txin := vm.tx.Inputs[vm.inputIndex]
	ii, ok := txin.TypedInput.(*bc.IssuanceInput)
	if !ok {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	return vm.push(ii.Nonce, true)
}

func opNextProgram(vm *virtualMachine) error {
	if vm.block == nil {
		return ErrContext
	}
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	return vm.push(vm.block.ConsensusProgram, true)
}

func opBlockTime(vm *virtualMachine) error {
	if vm.block == nil {
		return ErrContext
	}
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	if vm.block.TimestampMS > math.MaxInt64 {
		return fmt.Errorf("block timestamp out of range")
	}
	return vm.pushInt64(int64(vm.block.TimestampMS), true)
}
