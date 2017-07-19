package ca

import "chain/crypto/ed25519/ecmath"

type AssetRangeProof struct {
	Commitments []*AssetCommitment
	Signature   *RingSignature
	AssetID     *AssetID // nil means "confidential"
}

// CreateAssetRangeProof creates a confidential asset range proof. The
// caller can decorate the result with an asset ID to make it
// non-confidential.
func CreateAssetRangeProof(msg []byte, ac []*AssetCommitment, acPrime *AssetCommitment, j uint64, c, cPrime ecmath.Scalar) *AssetRangeProof {
	msghash := arpMsgHash(msg, ac, acPrime)
	P := arpPubkeys(ac, acPrime)
	var p ecmath.Scalar
	p.Sub(&cPrime, &c)
	rs := CreateRingSignature(msghash, []ecmath.Point{G, J}, P, j, p)
	return &AssetRangeProof{
		Commitments: ac,
		Signature:   rs,
	}
}

func (arp *AssetRangeProof) Validate(msg []byte, acPrime *AssetCommitment) bool {
	msghash := arpMsgHash(msg, arp.Commitments, acPrime)
	P := arpPubkeys(arp.Commitments, acPrime)
	if !arp.Signature.Validate(msghash, []ecmath.Point{G, J}, P) {
		return false
	}
	if arp.AssetID != nil {
		if !acPrime[1].ConstTimeEqual(&ecmath.ZeroPoint) {
			return false
		}
		assetPoint := CreateAssetPoint(arp.AssetID)
		return acPrime.H().ConstTimeEqual((*ecmath.Point)(&assetPoint))
	}
	return true
}

func arpMsgHash(msg []byte, ac []*AssetCommitment, acPrime *AssetCommitment) []byte {
	// msghash = Hash256("ARP.msg", AC’, AC[0], ..., AC[n-1], message)
	hasher := hasher256("ChainCA.ARP.msg", acPrime.Bytes())
	for _, aci := range ac {
		hasher.WriteItem(aci.Bytes())
	}
	hasher.Write(msg)
	var result [32]byte
	hasher.Sum(result[:0])
	return result[:]
}

func arpPubkeys(ac []*AssetCommitment, acPrime *AssetCommitment) [][]ecmath.Point {
	n := len(ac)
	result := make([][]ecmath.Point, n)
	for i := 0; i < n; i++ {
		var pp PointPair
		pp.Sub((*PointPair)(acPrime), (*PointPair)(ac[i]))
		result[i] = pp[:]
	}
	return result
}
