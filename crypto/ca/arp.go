package ca

import "chain/crypto/ed25519/ecmath"

type AssetRangeProof struct {
	commitments []*AssetCommitment
	signature   *RingSignature
	id          *AssetID // nil means "confidential"
}

// CreateAssetRangeProof creates a confidential asset range proof. The
// caller can decorate the result with an asset ID to make it
// non-confidential.
func CreateAssetRangeProof(msg []byte, ac []*AssetCommitment, acPrime *AssetCommitment, j uint64, c, cPrime ecmath.Scalar) *AssetRangeProof {
	P := arpPubkeys(ac, acPrime)
	var p ecmath.Scalar
	p.Sub(&cPrime, &c)
	rs := CreateRingSignature(msg, []ecmath.Point{G, J}, P, j, p)
	return &AssetRangeProof{
		commitments: ac,
		signature:   rs,
	}
}

func (arp *AssetRangeProof) Validate(msg []byte, acPrime *AssetCommitment) bool {
	// xxx pending: whether/how to hash msg before calling
	// arp.signature.Validate, which also hashes
	P := arpPubkeys(arp.commitments, acPrime)
	if !arp.signature.Validate(msg, []ecmath.Point{G, J}, P) {
		return false
	}
	if arp.id != nil {
		if !acPrime[1].ConstTimeEqual(&ecmath.ZeroPoint) {
			return false
		}
		assetPoint := CreateAssetPoint(arp.id)
		return acPrime[0].ConstTimeEqual((*ecmath.Point)(&assetPoint))
	}
	return true
}

func arpMsgHash(msg []byte, ac []*AssetCommitment, acPrime *AssetCommitment) [32]byte {
	hasher := hasher256("ChainCA.ARP.msg", acPrime.Bytes())
	for _, aci := range ac {
		hasher.WriteItem(aci.Bytes())
	}
	hasher.Write(msg)
	var result [32]byte
	hasher.Sum(result[:0])
	return result
}

func arpPubkeys(ac []*AssetCommitment, acPrime *AssetCommitment) [][]ecmath.Point {
	n := len(ac)
	result := make([][]ecmath.Point, n)
	for i := 0; i < n; i++ {
		result[i] = make([]ecmath.Point, 2)
		result[i][0].Sub(&acPrime[0], &ac[i][0])
		result[i][1].Sub(&acPrime[1], &ac[i][1])
	}
	return result
}
