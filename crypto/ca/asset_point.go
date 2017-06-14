package ca

import "chain/crypto/ed25519/ecmath"

// AssetPoint is an orthogonal generator to which an AssetID is mapped.
type AssetPoint ecmath.Point

// CreateAssetPoint converts an asset ID to a valid orthogonal generator on EC.
func CreateAssetPoint(assetID AssetID) AssetPoint {
	return AssetPoint(pointHash("ChainCA.AssetID", assetID[:]))
}
