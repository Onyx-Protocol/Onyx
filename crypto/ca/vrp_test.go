package ca

import (
	"reflect"
	"testing"
)

func TestVRP(t *testing.T) {
	assetID := AssetID{1}
	aek := AssetKey{2}
	ac, _ := CreateAssetCommitment(assetID, aek)

	value := uint64(3)
	vek := ValueKey{4}
	vc, f := CreateValueCommitment(value, ac, vek)

	N := uint64(8)

	pt := make([][32]byte, 2*N-1)
	pt[10][9] = 'x'

	idek := DataKey{5}

	msg := []byte("message")

	vrp := CreateValueRangeProof(ac, vc, N, value, pt, *f, idek, vek, msg)
	if !vrp.Validate(ac, vc, msg) {
		t.Error("failed to validate vrp")
	}

	if vrp.Validate(ac, vc, msg[1:]) {
		t.Error("validated invalid vrp")
	}

	ac2 := *ac
	ac2[0].Add(&ac2[0], &G)
	if vrp.Validate(&ac2, vc, msg) {
		t.Error("validated invalid vrp")
	}
	ac2 = *ac
	ac2[1].Add(&ac2[1], &G)
	if vrp.Validate(&ac2, vc, msg) {
		t.Error("validated invalid vrp")
	}

	vc2 := *vc
	vc2[0].Add(&vc2[0], &G)
	if vrp.Validate(ac, &vc2, msg) {
		t.Error("validated invalid vrp")
	}
	vc2 = *vc
	vc2[1].Add(&vc2[1], &G)
	if vrp.Validate(ac, &vc2, msg) {
		t.Error("validated invalid vrp")
	}

	vrp2 := *vrp
	vrp2.exp++
	if vrp2.Validate(ac, vc, msg) {
		t.Error("validated invalid vrp")
	}
	vrp2 = *vrp
	vrp2.vmin++
	if vrp2.Validate(ac, vc, msg) {
		t.Error("validated invalid vrp")
	}

	pt2 := vrp.Payload(ac, vc, value, *f, idek, vek, msg)
	if !reflect.DeepEqual(pt, pt2) {
		t.Errorf("got %v, want %v", pt2, pt)
	}
}
