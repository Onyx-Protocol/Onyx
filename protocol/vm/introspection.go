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
	refdatahash, err := vm.pop(true)
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

	ok, err := vm.vmContext.CheckOutput(uint64(index), refdatahash, uint64(amount), assetID, uint64(vmVersion), code)
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

	return vm.push(vm.vmContext.Code(), true)
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

	data, err := vm.vmContext.InputRefDataHash()
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

	data, err := vm.vmContext.TxRefDataHash()
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

	inputIndex, err := vm.vmContext.InputIndex()
	if err != nil {
		return err
	}

	return vm.pushInt64(int64(inputIndex), true)
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

	nonce, err := vm.vmContext.Nonce()
	if err != nil {
		return err
	}

	return vm.push(nonce, true)
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

	timestampMS, err := vm.vmContext.BlockTimeMS()
	if err != nil {
		return err
	}

	return vm.pushInt64(int64(timestampMS), true)
}
