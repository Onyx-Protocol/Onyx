package ca

import "chain/crypto/ed25519/ecmath"

// AssetPoint is an orthogonal generator to which an AssetID is mapped.
type AssetPoint ecmath.Point

// CreateAssetPoint converts an asset ID to a valid orthogonal generator on EC.
func CreateAssetPoint(assetID AssetID) AssetPoint {
	var P ecmath.Point

	for ctr := uint64(0); true; ctr++ {
		// 1. Calculate `Hash256("AssetID" || assetID || uint64le(counter))`
		h := hash256([]byte("AssetID"), assetID[:], uint64le(ctr))

		// 2. Decode the resulting hash as a point `P` on the elliptic curve.
		err := P.UnmarshalBinary(h[:])

		if err != nil {
			continue
		}

		// 3. Calculate point `A = 8*P` (8 is a cofactor in edwards25519) which belongs to a subgroup of `G` with order `L`.
		cofactor := ecmath.Scalar{8}
		P.ScMul(&P, &cofactor)

		break
	}
	// 4. Return `A`.
	return AssetPoint(P)
}
