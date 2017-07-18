package txvm2

import (
	"fmt"

	"chain/crypto/sha3pool"
	"chain/protocol/bc"
)

func opSummarize(vm *vm) {
	if vm.summarized {
		panic("summarize: already summarized")
	}

	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	for _, item := range vm.stacks[effectstack] {
		item.encode(hasher)
	}
	var h [32]byte
	hasher.Read(h[:])

	vm.push(effectstack, mkSummary(vm.txVersion, vm.txRunlimit, h[:]))

	vm.summarized = true
}

func opUnlockLegacy(vm *vm) {
	outTuple := vm.popTuple(datastack, legacyOutputTuple)
	var sourceHashBytes [32]byte
	copy(sourceHashBytes[:], legacyOutputSourceID(outTuple))
	sourceHash := bc.NewHash(sourceHashBytes)
	var assetIDBytes [32]byte
	copy(assetIDBytes[:], legacyOutputAssetID(outTuple))
	assetID := bc.NewAssetID(assetIDBytes)
	amount := legacyOutputAmount(outTuple)
	assetAmount := &bc.AssetAmount{
		AssetId: &assetID,
		Amount:  uint64(amount),
	}
	source := &bc.ValueSource{
		Ref:      &sourceHash,
		Value:    assetAmount,
		Position: uint64(legacyOutputIndex(outTuple)), // xxx check this is the right use for `index`
	}
	var hashBytes [32]byte
	copy(hashBytes[:], legacyOutputData(outTuple))
	data := bc.NewHash(hashBytes)
	out := bc.NewOutput(source, &bc.Program{VmVersion: 1, Code: legacyOutputProgram(outTuple)}, &data, 0) // xxx check ordinal of 0 is ok
	outID := bc.EntryID(out)
	vm.push(entrystack, mkInput(outID.Bytes()))
	vm.push(entrystack, mkAnchor(outID.Bytes()))
	vm.push(entrystack, mkValue(amount, assetIDBytes[:]))
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
	s := vm.stacks[stackID]
	if n >= vint64(len(s)) {
		panic(fmt.Errorf("extend: stack offset %d greater than %d-item stack", n, len(s)))
	}
	t, ok := s[n].(tuple)
	if !ok {
		panic(fmt.Errorf("extend: item %d on stack %d is a %T, not a tuple", n, stackID, s[n]))
	}
	t = append(t, item)
	s[n] = t
}
