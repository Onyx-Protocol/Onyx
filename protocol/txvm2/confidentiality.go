package txvm2

import (
	"fmt"

	"chain/crypto/ca"
)

func opWrapValue(vm *vm) {
	val := vm.popValue(entrystack)
	ac, vc := val.commitments()
	vm.pushAssetcommitment(entrystack, ac)
	vm.pushValuecommitment(entrystack, vc)
}

func opMergeConfidential(vm *vm) {
	a := vm.popTuple(entrystack, valueType, provenvalueType, unprovenvalueType)
	b := vm.popTuple(entrystack, valueType, provenvalueType, unprovenvalueType)

	_, vca := toCommitments(a)
	_, vcb := toCommitments(b)

	vca.Add(vca, vcb)
	vc := valuecommitment{vca}
	vm.pushUnprovenvalue(entrystack, unprovenvalue{vc})
}

func opSplitConfidential(vm *vm) {
	val := vm.popTuple(entrystack, valueType, provenvalueType, unprovenvalueType)
	_, orig := toCommitments(val)

	split := vm.popValuecommitment(entrystack)

	vm.pushUnprovenvalue(entrystack, unprovenvalue{split})

	var diff ca.ValueCommitment

	diff.Sub(orig, split.vc)
	diffvc := valuecommitment{&diff}
	vm.pushUnprovenvalue(entrystack, unprovenvalue{diffvc})
}

func opProveAssetRange(vm *vm) {
	rsBytes := vm.popBytes(datastack)
	var rs ca.RingSignature
	ok := rs.Decode(rsBytes)
	if !ok {
		panic(fmt.Errorf("proveassetrange: bad ring signature length %d", len(rsBytes)))
	}

	n := int64(len(rsBytes)/32 - 1)

	prog := vm.popBytes(datastack)

	ac := vm.popAssetcommitment(datastack)

	items := vm.peekN(entrystack, n)
	var acs []*ca.AssetCommitment
	for _, t := range items {
		var ac assetcommitment
		if !ac.detuple(t.(tuple)) {
			panic(fmt.Errorf("%T on entry stack is not an assetcommitment", t))
		}
		acs = append(acs, ac.ac)
	}

	arp := &ca.AssetRangeProof{
		Commitments: acs,
		Signature:   &rs,
	}
	if !arp.Validate(prog, ac.ac) {
		panic("invalid asset range proof")
	}

	vm.pushAssetcommitment(entrystack, ac)
	doCommand(vm, prog)
}

func opDropAssetCommitment(vm *vm) {
	// xxx
}

func opProveAssetID(vm *vm) {
	// xxx
}

func opProveAmount(vm *vm) {
	// xxx
}

func opProveValueRange(vm *vm) {
	// xxx
}

func opIssuanceCandidate(vm *vm) {
	// xxx
}

func opIssueConfidential(vm *vm) {
	// xxx
}

func (v *value) commitments() (assetcommitment, valuecommitment) {
	var assetID ca.AssetID
	copy(assetID[:], v.assetID)
	ac, _ := ca.CreateAssetCommitment(assetID, nil)
	vc, _ := ca.CreateValueCommitment(uint64(v.amount), ac, nil)
	return assetcommitment{ac}, valuecommitment{vc}
}

func toCommitments(t namedtuple) (*ca.AssetCommitment, *ca.ValueCommitment) {
	switch tt := t.(type) {
	case value:
		ac, vc := tt.commitments()
		return ac.ac, vc.vc

	case assetcommitment:
		return tt.ac, nil

	case valuecommitment:
		return nil, tt.vc

	case provenvalue:
		return tt.ac.ac, tt.vc.vc

	case unprovenvalue:
		return nil, tt.vc.vc
	}
	return nil, nil
}
