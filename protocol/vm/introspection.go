package vm

import "math"

func opCheckOutput(vm *virtualMachine) error {
	err := vm.applyCost(16)
	if err != nil {
		return err
	}

	code, err := vm.pop(true)
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
	data, err := vm.pop(true)
	if err != nil {
		return err
	}
	index, err := vm.popInt64(true)
	if err != nil {
		return err
	}
	if index < 0 {
		return ErrBadValue
	}

	if vm.context.CheckOutput == nil {
		return ErrContext
	}

	ok, err := vm.context.CheckOutput(uint64(index), data, uint64(amount), assetID, uint64(vmVersion), code, vm.expansionReserved)
	if err != nil {
		return err
	}
	return vm.pushBool(ok, true)
}

func opAsset(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.AssetID == nil {
		return ErrContext
	}
	return vm.push(*vm.context.AssetID, true)
}

func opAmount(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.Amount == nil {
		return ErrContext
	}
	return vm.pushInt64(int64(*vm.context.Amount), true)
}

func opProgram(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	return vm.push(vm.context.Code, true)
}

func opMinTime(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.MinTimeMS == nil {
		return ErrContext
	}
	return vm.pushInt64(int64(*vm.context.MinTimeMS), true)
}

func opMaxTime(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.MaxTimeMS == nil {
		return ErrContext
	}
	maxTimeMS := *vm.context.MaxTimeMS
	if maxTimeMS == 0 || maxTimeMS > math.MaxInt64 {
		maxTimeMS = uint64(math.MaxInt64)
	}

	return vm.pushInt64(int64(maxTimeMS), true)
}

func opEntryData(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.EntryData == nil {
		return ErrContext
	}

	return vm.push(*vm.context.EntryData, true)
}

func opTxData(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.TxData == nil {
		return ErrContext
	}
	return vm.push(*vm.context.TxData, true)
}

func opIndex(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.DestPos == nil {
		return ErrContext
	}
	return vm.pushInt64(int64(*vm.context.DestPos), true)
}

func opEntryID(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	return vm.push(vm.context.EntryID, true)
}

func opOutputID(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.SpentOutputID == nil {
		return ErrContext
	}
	return vm.push(*vm.context.SpentOutputID, true)
}

func opNonce(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.AnchorID == nil {
		return ErrContext
	}
	return vm.push(*vm.context.AnchorID, true)
}

func opNextProgram(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.NextConsensusProgram == nil {
		return ErrContext
	}
	return vm.push(*vm.context.NextConsensusProgram, true)
}

func opBlockTime(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	if vm.context.BlockTimeMS == nil {
		return ErrContext
	}
	return vm.pushInt64(int64(*vm.context.BlockTimeMS), true)
}
