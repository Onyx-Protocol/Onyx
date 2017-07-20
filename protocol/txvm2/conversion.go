package txvm2

import (
	"fmt"

	"chain/crypto/sha3pool"
	"chain/protocol/bc"
)

func opFinalize(vm *vm) {
	if vm.finalized {
		panic("finalize: already finalized")
	}

	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	s := vm.getStack(effectstack)
	for _, item := range *s {
		item.encode(hasher)
	}
	var h [32]byte
	hasher.Read(h[:])

	vm.push(effectstack, mkTransaction(vint64(vm.txVersion), vint64(vm.initialRunlimit), h[:]))

	vm.finalized = true
}

func opUnlockLegacy(vm *vm) {
	leg := vm.popLegacyoutput(datastack)
	var sourceHashBytes [32]byte
	copy(sourceHashBytes[:], leg.sourceID)
	sourceHash := bc.NewHash(sourceHashBytes)
	var assetIDBytes [32]byte
	copy(assetIDBytes[:], leg.assetID)
	assetID := bc.NewAssetID(assetIDBytes)
	assetAmount := &bc.AssetAmount{
		AssetId: &assetID,
		Amount:  uint64(leg.amount), // xxx check leg.amount >= 0?
	}
	source := &bc.ValueSource{
		Ref:      &sourceHash,
		Value:    assetAmount,
		Position: uint64(leg.index), // xxx check this is the right use for `index`
	}
	var hashBytes [32]byte
	copy(hashBytes[:], leg.data)
	data := bc.NewHash(hashBytes)
	out := bc.NewOutput(source, &bc.Program{VmVersion: 1, Code: leg.program}, &data, 0) // xxx check ordinal of 0 is ok
	outID := bc.EntryID(out).Bytes()
	vm.pushInput(entrystack, input{outID})
	vm.pushAnchor(entrystack, anchor{outID})
	vm.pushValue(entrystack, value{leg.amount, leg.assetID})
	// xxx something something something legacy control program (deferred run with vm1, or translation to txvm)
}

func opIssueLegacy(vm *vm) {
	// xxx
}

func opLegacyIssuanceCandidate(vm *vm) {
	// xxx
}

func opExtend(vm *vm) {
	// xxx check extension flag
	stackID := vm.popInt64(datastack)
	n := vm.popInt64(datastack)
	if n < 0 {
		panic(fmt.Errorf("extend: negative stack offset %d", n))
	}
	item := vm.pop(datastack)
	s := vm.getStack(int64(stackID))
	if n >= int64(len(*s)) {
		panic(fmt.Errorf("extend: stack offset %d greater than %d-item stack", n, len(*s)))
	}
	t, ok := (*s)[n].(tuple)
	if !ok {
		panic(fmt.Errorf("extend: item %d on stack %d is a %T, not a tuple", n, stackID, (*s)[n]))
	}
	t = append(t, item)
	(*s)[n] = t
}
