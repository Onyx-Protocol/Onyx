package txvm2

import (
	"fmt"

	"chain/crypto/ca"
	"chain/crypto/ed25519/ecmath"
)

func (v *value) commitments() (assetcommitment, valuecommitment) {
	var assetID ca.AssetID
	copy(assetID[:], v.assetID)
	ac, _ := ca.CreateAssetCommitment(assetID, nil)
	vc, _ := ca.CreateValueCommitment(uint64(v.amount), ac, nil)
	return assetcommitment{ac}, valuecommitment{vc}
}

func opWrapValue(vm *vm) {
	val := vm.popValue(entrystack)
	ac, vc := val.commitments()
	vm.pushAssetcommitment(entrystack, ac)
	vm.pushValuecommitment(entrystack, vc)
}

func opMergeConfidential(vm *vm) {
	a := vm.popTuple(entrystack, valueTuple, provenValueTuple, unprovenValueTuple)
	b := vm.popTuple(entrystack, valueTuple, provenValueTuple, unprovenValueTuple)

	_, vca := tupleToCommitments(a)
	_, vcb := tupleToCommitments(b)

	vca.Add(vca, vcb)
	vc := valuecommitment{vca}
	vm.pushUnprovenvalue(entrystack, unprovenvalue{vc})
}

func opSplitConfidential(vm *vm) {
	val := vm.popTuple(entrystack, valueTuple, provenValueTuple, unprovenValueTuple)
	_, orig := tupleToCommitments(val)

	split := vm.popValuecommitment(entrystack)

	vm.push(entrystack, mkUnprovenValue(split.entuple()))

	var diff ca.ValueCommitment

	diff.Sub(orig, split.vc)
	difftuple := mkValueCommitment(vbytes(diff.V().Bytes()), vbytes(diff.F().Bytes()))
	vm.push(entrystack, mkUnprovenValue(difftuple))
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

	prevacTuples := vm.peekNTuple(entrystack, n, assetCommitmentTuple)

	var prevacs []*ca.AssetCommitment
	for _, t := range prevacTuples {
		prevac, _ := tupleToCommitments(t)
		prevacs = append(prevacs, prevac)
	}

	arp := &ca.AssetRangeProof{
		Commitments: prevacs,
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

func tupleToCommitments(t tuple) (*ca.AssetCommitment, *ca.ValueCommitment) {
	name, ok := t.name()
	if !ok {
		return nil, nil
	}
	var atuple, vtuple tuple
	switch name {
	case valueTuple:
		var assetID ca.AssetID
		copy(assetID[:], valueAssetID(t))
		ac, _ := ca.CreateAssetCommitment(assetID, nil)
		vc, _ := ca.CreateValueCommitment(uint64(valueAmount(t)), ac, nil)
		return ac, vc

	case assetCommitmentTuple:
		atuple = t

	case valueCommitmentTuple:
		vtuple = t

	case provenValueTuple:
		atuple = provenValueAssetCommitment(t)
		vtuple = provenValueValueCommitment(t)

	case unprovenValueTuple:
		vtuple = unprovenValueValueCommitment(t)

	default:
		return nil, nil
	}
	var ac *ca.AssetCommitment
	if atuple != nil {
		var (
			H, C ecmath.Point
			buf  [32]byte
		)
		copy(buf[:], assetCommitmentAssetPoint(atuple))
		_, ok := H.Decode(buf)
		if !ok {
			return nil, nil
		}
		copy(buf[:], assetCommitmentBlindingPoint(atuple))
		_, ok = C.Decode(buf)
		if !ok {
			return nil, nil
		}
		ac = &ca.AssetCommitment{H, C}
	}
	var vc *ca.ValueCommitment
	if vtuple != nil {
		var (
			V, F ecmath.Point
			buf  [32]byte
		)
		copy(buf[:], valueCommitmentValuePoint(vtuple))
		_, ok := V.Decode(buf)
		if !ok {
			return nil, nil
		}
		copy(buf[:], valueCommitmentBlindingPoint(vtuple))
		_, ok = F.Decode(buf)
		if !ok {
			return nil, nil
		}
		vc = &ca.ValueCommitment{V, F}
	}
	return ac, vc
}

func tupleFromVC(vc *ca.ValueCommitment) tuple {
	return mkValueCommitment(vbytes(vc.V().Bytes()), vbytes(vc.F().Bytes()))
}
