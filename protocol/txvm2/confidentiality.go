package txvm2

import (
	"fmt"

	"chain/crypto/ca"
	"chain/crypto/ed25519/ecmath"
)

// inp is a value tuple
func wrapvalue(inp tuple) (a, v tuple) {
	ac, vc := tupleToCommitments(inp)
	a = mkAssetCommitment(vbytes(ac.H().Bytes()), vbytes(ac.C().Bytes()))
	v = mkValueCommitment(vbytes(vc.V().Bytes()), vbytes(vc.F().Bytes()))
	return a, v
}

func opWrapValue(vm *vm) {
	val := vm.popTuple(entrystack, valueTuple)
	ac, vc := wrapvalue(val)
	vm.push(entrystack, ac)
	vm.push(entrystack, vc)
}

func opMergeConfidential(vm *vm) {
	a := vm.popTuple(entrystack, valueTuple, provenValueTuple, unprovenValueTuple)
	b := vm.popTuple(entrystack, valueTuple, provenValueTuple, unprovenValueTuple)

	getValueCommitment := func(a tuple) *ca.ValueCommitment {
		name, _ := a.name()
		if name == valueTuple {
			var assetID ca.AssetID
			copy(assetID[:], valueAssetID(a))
			ac, _ := ca.CreateAssetCommitment(assetID, nil)
			vc, _ := ca.CreateValueCommitment(uint64(valueAmount(a)), ac, nil)
			return vc
		}
		var vctuple tuple
		if name == provenValueTuple {
			vctuple = provenValueValueCommitment(a)
		} else {
			vctuple = unprovenValueValueCommitment(a)
		}
		var V, F ecmath.Point
		var pointBytes [32]byte
		copy(pointBytes[:], valueCommitmentValuePoint(vctuple))
		_, ok := V.Decode(pointBytes)
		if !ok {
			panic("mergeconfidential: invalid curve point")
		}
		copy(pointBytes[:], valueCommitmentBlindingPoint(vctuple))
		_, ok = F.Decode(pointBytes)
		if !ok {
			panic("mergeconfidential: invalid curve point")
		}
		return &ca.ValueCommitment{V, F}
	}

	vca := getValueCommitment(a)
	vcb := getValueCommitment(b)
	vca.Add(vca, vcb)
	vcbytes := vca.Bytes()
	vc := mkValueCommitment(vbytes(vcbytes[:32]), vbytes(vcbytes[32:]))
	vm.push(entrystack, mkUnprovenValue(vc))
}

func opSplitConfidential(vm *vm) {
	val := vm.popTuple(entrystack, valueTuple, provenValueTuple, unprovenValueTuple)
	_, orig := tupleToCommitments(val)

	vctuple := vm.popTuple(entrystack, valueCommitmentTuple)
	_, split := tupleToCommitments(vctuple)

	vm.push(entrystack, mkUnprovenValue(vctuple))

	var diff ca.ValueCommitment

	diff.Sub(orig, split)
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

	acTuple := vm.popTuple(datastack, assetCommitmentTuple)

	prevacTuples := vm.peekNTuple(entrystack, n, assetCommitmentTuple)

	ac, _ := tupleToCommitments(acTuple)

	var prevacs []*ca.AssetCommitment
	for _, t := range prevacTuples {
		prevac, _ := tupleToCommitments(t)
		prevacs = append(prevacs, prevac)
	}

	arp := &ca.AssetRangeProof{
		Commitments: prevacs,
		Signature:   &rs,
	}
	if !arp.Validate(prog, ac) {
		panic("invalid asset range proof")
	}

	vm.push(entrystack, acTuple)
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
