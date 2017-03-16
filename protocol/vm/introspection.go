package vm

import (
	"bytes"
	"fmt"
	"math"

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

	// The following is per the discussion at
	// https://chainhq.slack.com/archives/txgraph/p1487964172000960
	var inpDest bc.ValueDestination
	switch inp := vm.tx.TxInputs[vm.inputIndex].(type) {
	case *bc.Spend:
		inpDest = inp.Destination()
	case *bc.Issuance:
		inpDest = inp.Destination()
	default:
		return ErrContext // xxx ?
	}
	mux, ok := inpDest.Entry.(*bc.Mux)
	if !ok {
		return vm.pushBool(false, true)
	}
	muxDests := mux.Destinations()
	if index >= int64(len(muxDests)) {
		return ErrBadValue // xxx or simply return false?
	}

	someChecks := func(resAssetID bc.AssetID, resAmount uint64, resData bc.Hash) bool {
		if !bytes.Equal(resAssetID[:], assetID) {
			return false
		}
		if resAmount != uint64(amount) {
			return false
		}
		if len(refdatahash) > 0 && !bytes.Equal(refdatahash, resData[:]) {
			return false
		}
		return true
	}

	if vmVersion == 1 && len(code) > 0 && code[0] == byte(OP_FAIL) {
		// Special case alert! Old-style retirements were just outputs
		// with a control program beginning [FAIL]. New-style retirements
		// do not have control programs, but for compatibility we allow
		// CHECKOUTPUT to test for them by specifying a programming
		// beginnning with [FAIL].
		r, ok := muxDests[index].Entry.(*bc.Retirement)
		if !ok {
			return vm.pushBool(false, true)
		}
		ok = someChecks(r.AssetID(), r.Amount(), r.Data())
		return vm.pushBool(ok, true)
	}

	o, ok := muxDests[index].Entry.(*bc.Output)
	if !ok {
		return vm.pushBool(false, true)
	}

	if !someChecks(o.AssetID(), o.Amount(), o.Data()) {
		return vm.pushBool(false, true)
	}
	prog := o.ControlProgram()
	if prog.VMVersion != uint64(vmVersion) {
		return vm.pushBool(false, true)
	}
	if !bytes.Equal(prog.Code, code) {
		return vm.pushBool(false, true)
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

	var assetID bc.AssetID
	switch inp := vm.tx.TxInputs[vm.inputIndex].(type) {
	case *bc.Issuance:
		assetID = inp.AssetID()
	case *bc.Spend:
		assetID = inp.AssetID()
	default:
		return ErrContext // xxx right?
	}

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

	var amount uint64
	switch inp := vm.tx.TxInputs[vm.inputIndex].(type) {
	case *bc.Issuance:
		amount = inp.Amount()
	case *bc.Spend:
		amount = inp.Amount()
	default:
		return ErrContext // xxx ?
	}

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

	return vm.push(vm.mainprog, true)
}

func opMinTime(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	return vm.pushInt64(int64(vm.tx.MinTimeMS()), true)
}

func opMaxTime(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	maxTime := vm.tx.MaxTimeMS()
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

	var data bc.Hash
	switch inp := vm.tx.TxInputs[vm.inputIndex].(type) {
	case *bc.Issuance:
		data = inp.Data()
	case *bc.Spend:
		data = inp.Data()
	default:
		return ErrContext // xxx ?
	}

	return vm.push(data[:], true)
}

func opTxRefDataHash(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	h := vm.tx.Data()
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

func opOutputID(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	sp, ok := vm.tx.TxInputs[vm.inputIndex].(*bc.Spend)
	if !ok {
		return ErrContext
	}
	outid := sp.SpentOutputID()
	return vm.push(outid[:], true)
}

func opNonce(vm *virtualMachine) error {
	if vm.tx == nil {
		return ErrContext
	}

	err := vm.applyCost(1)
	if err != nil {
		return err
	}

	iss, ok := vm.tx.TxInputs[vm.inputIndex].(*bc.Issuance)
	if !ok {
		return ErrContext
	}

	anchorID := iss.AnchorID() // xxx right?
	return vm.push(anchorID[:], true)
}

func opNextProgram(vm *virtualMachine) error {
	if vm.block == nil {
		return ErrContext
	}
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	return vm.push(vm.block.NextConsensusProgram(), true)
}

func opBlockTime(vm *virtualMachine) error {
	if vm.block == nil {
		return ErrContext
	}
	err := vm.applyCost(1)
	if err != nil {
		return err
	}
	if vm.block.TimestampMS() > math.MaxInt64 {
		return fmt.Errorf("block timestamp out of range")
	}
	return vm.pushInt64(int64(vm.block.TimestampMS()), true)
}
