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
	numResults, err := vm.vmContext.NumResults()
	if err != nil {
		return err
	}
	if index < 0 || index >= numResults {
		return ErrBadValue
	}
	ok, err = vm.vmContext.CheckOutput(index, data, amount, assetID, vmVersion, code)
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

	assetID, err := vm.vmContext.AssetID()
	if err != nil {
		return err
	}

	return vm.push(assetID, true)
}

func opAmount(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	amount, err := vm.vmContext.Amount()
	if err != nil {
		return err
	}

	return vm.pushInt64(int64(amount), true)
}

func opProgram(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	return vm.push(vm.mainprog, true)
}

func opMinTime(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	minTimeMS, err := vm.vmContext.MinTimeMS()
	if err != nil {
		return err
	}

	return vm.pushInt64(int64(minTimeMS), true)
}

func opMaxTime(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	maxTimeMS, err := vm.vmContext.MaxTimeMS()
	if maxTimeMS == 0 || maxTimeMS > math.MaxInt64 {
		maxTimeMS = uint64(math.MaxInt64)
	}

	return vm.pushInt64(int64(maxTimeMS), true)
}

func opRefDataHash(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	data, err := vm.vmContext.EntryData()
	if err != nil {
		return err
	}

	return vm.push(data, true)
}

func opTxRefDataHash(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	data, err := vm.vmContext.TxData()
	if err != nil {
		return err
	}

	return vm.push(data, true)
}

func opIndex(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	destPos, err := vm.vmContext.DestPos()
	if err != nil {
		return err
	}

	return vm.pushInt64(int64(destPos), true)
}

func opOutputID(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	spentOutputID, err := vm.vmContext.SpentOutputID()
	if err != nil {
		return err
	}
	return vm.push(spentOutputID, true)
}

func opNonce(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	anchorID, err := vm.vmContext.AnchorID()
	if err != nil {
		return err
	}

	return vm.push(anchorID, true)
}

func opNextProgram(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	prog, err := vm.vmContext.NextConsensusProgram()
	if err != nil {
		return err
	}

	return vm.push(prog, true)
}

func opBlockTime(vm *virtualMachine) error {
	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	timestampMS, err := vm.vmContext.BlockTime()
	if err != nil {
		return err
	}

	return vm.pushInt64(int64(timestampMS), true)
}
