package ca

import (
	"bytes"

	"chain/crypto/ed25519/ecmath"
	"chain/encoding/blockchain"
)

type EncryptedOutput struct {
	ac  *AssetCommitment
	vc  *ValueCommitment
	vrp *ValueRangeProof
}

func EncryptOutput(assetID AssetID, value uint64, N uint64, plaintext []byte, q *ecmath.Scalar, aek AssetKey, vek ValueKey, idek DataKey) (eo *EncryptedOutput, c, f *ecmath.Scalar) {
	if value >= 1<<N {
		return nil, nil, nil // xxx or panic
	}
	ptbuf := new(bytes.Buffer)
	blockchain.WriteVarstr31(ptbuf, plaintext)
	if uint64(ptbuf.Len()) > 32*(2*N-1) {
		return nil, nil, nil // xxx or panic
	}
	pt := make([][32]byte, 2*N-1)
	for i := 0; ptbuf.Len() > 0; i++ {
		ptbuf.Read(pt[i][:]) // xxx check err
	}
	ac, c := CreateAssetCommitment(assetID, aek)
	vc, f := CreateValueCommitment(value, ac, vek)
	if q != nil {
		extra := *q
		extra.Sub(&extra, f)
		var vscalar ecmath.Scalar
		vscalar.SetUint64(value)
		vscalar.Mul(&vscalar, c)
		extra.Sub(&extra, &vscalar) // extra = q - f - value·c
		f.Add(f, &extra)
		GJ := &PointPair{G, J}
		GJ.ScMul(GJ, &extra)
		(*PointPair)(vc).Add((*PointPair)(vc), GJ)
	}
	vrp := CreateValueRangeProof(ac, vc, N, value, pt, *f, idek, vek, nil) // xxx nil or msg?
	return &EncryptedOutput{ac: ac, vc: vc, vrp: vrp}, c, f
}

func (eo *EncryptedOutput) Decrypt(arp *AssetRangeProof, evef *EncryptedValue, eaec *EncryptedAssetID, aek AssetKey, vek ValueKey, idek DataKey) (assetID AssetID, value uint64, c, f ecmath.Scalar, plaintext []byte, ok bool) {
	if arp.id == nil {
		// Confidential asset range proof
		assetID, c, ok = eaec.Decrypt(eo.ac, aek)
		if !ok {
			return
		}
	} else {
		// Non-confidential asset range proof
		assetID = *arp.id
		c = ecmath.Zero
	}

	// xxx handle non-confidential vrp
	value, f, ok = evef.Decrypt(eo.vc, eo.ac, vek)
	if !ok {
		return
	}
	pt := eo.vrp.Payload(eo.ac, eo.vc, value, f, idek, vek, nil) // xxx nil or msg?
	buf := new(bytes.Buffer)
	for _, pti := range pt {
		buf.Write(pti[:])
	}
	plaintext, _ = blockchain.ReadVarstr31(buf) // xxx check err
	return assetID, value, c, f, plaintext, true
}
