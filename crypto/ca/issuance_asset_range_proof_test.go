package ca

import (
	"bytes"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestIssuanceAssetRangeProof(t *testing.T) {
	// 1. Assets and issuance keys

	apples := AssetID{0}
	oranges := AssetID{1}
	bananas := AssetID{2}

	assetIDs := []AssetID{
		apples,
		oranges,
		bananas,
	}
	A := []AssetCommitment{
		CreateNonblindedAssetCommitment(assetIDs[0]),
		CreateNonblindedAssetCommitment(assetIDs[1]),
		CreateNonblindedAssetCommitment(assetIDs[2]),
	}

	y := []Scalar{
		reducedScalar(hash512([]byte("apples"))),
		reducedScalar(hash512([]byte("oranges"))),
		reducedScalar(hash512([]byte("bananas"))),
	}
	Y := []Point{
		multiplyBasePoint(y[0]),
		multiplyBasePoint(y[1]),
		multiplyBasePoint(y[2]),
	}

	// 2. Encryption keys
	rek := RecordKey{1}
	aek := DeriveAssetKey(DeriveIntermediateKey(rek))
	vmver := uint64(1)
	program := []byte("allow this transaction")

	// 3. Encrypt each of assets and try to make a range proof
	for j := 0; j < len(A); j++ {
		H, c := CreateBlindedAssetCommitment(A[j], ZeroScalar, aek)

		// 1-item ring:
		rp, err := CreateIssuanceAssetRangeProof(
			H,
			c,
			assetIDs[j:j+1],
			Y[j:j+1],
			vmver,
			program,
			0,
			y[j],
		)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		err = rp.WriteTo(&buf)
		if err != nil {
			t.Fatal(err)
		}
		rp2 := new(IssuanceAssetRangeProof)
		err = rp2.ReadFrom(bytes.NewReader(buf.Bytes()), rp.rs.nPubkeys())
		if err != nil {
			t.Fatal(err)
		}
		if !iarpsEqual(rp, rp2) {
			t.Errorf("serialization failure. original:\n%s\ndeserialized:\n%s\nserialization: %x", spew.Sdump(rp), spew.Sdump(rp2), buf.Bytes())

			var y1, y2 bytes.Buffer

			rp.Y[0].WriteTo(&y1)
			rp2.Y[0].WriteTo(&y2)

			t.Logf("also, rp.Y[0] is %x, rp2.Y[0] is %x\n", y1.Bytes(), y2.Bytes())
		}
		if err = rp2.Verify(H, assetIDs[j:j+1]); err != nil {
			t.Errorf("Failed to verify range proof for asset %d: %s", j, err)
		}

		// 3-item ring:
		rp, err = CreateIssuanceAssetRangeProof(
			H,
			c,
			assetIDs,
			Y,
			vmver,
			program,
			j,
			y[j],
		)
		if err != nil {
			t.Fatal(err)
		}
		buf.Reset()
		err = rp.WriteTo(&buf)
		if err != nil {
			t.Fatal(err)
		}
		rp2 = new(IssuanceAssetRangeProof)
		err = rp2.ReadFrom(bytes.NewReader(buf.Bytes()), rp.rs.nPubkeys())
		if err != nil {
			t.Fatal(err)
		}
		if !iarpsEqual(rp, rp2) {
			t.Errorf("serialization failure. original:\n%s\ndeserialized:\n%s\nserialization: %x", spew.Sdump(rp), spew.Sdump(rp2), buf.Bytes())
		}
		if err = rp2.Verify(H, assetIDs); err != nil {
			t.Errorf("Failed to verify range proof for asset %d: %s", j, err)
		}
	}
}

func iarpsEqual(iarp1, iarp2 *IssuanceAssetRangeProof) bool {
	if !ringsigsEqual(iarp1.rs, iarp2.rs) {
		return false
	}
	if len(iarp1.Y) != len(iarp2.Y) {
		return false
	}
	for i, y := range iarp1.Y {
		if encodePoint(&y) != encodePoint(&iarp2.Y[i]) {
			return false
		}
	}
	if iarp1.vmver != iarp2.vmver {
		return false
	}
	if !bytes.Equal(iarp1.program, iarp2.program) {
		return false
	}
	if len(iarp1.args) != len(iarp2.args) {
		return false
	}
	for i, a := range iarp1.args {
		if !bytes.Equal(a, iarp2.args[i]) {
			return false
		}
	}
	return true
}

func init() {
	spew.Config.DisableMethods = true
	spew.Config.DisablePointerMethods = true
}
