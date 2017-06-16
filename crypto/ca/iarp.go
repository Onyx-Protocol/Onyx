package ca

import (
	"chain/crypto/ed25519/ecmath"
)

type IssuanceAssetRangeProof interface {
	// TODO: add Reader/Writer interfaces
	Validate(ac *AssetCommitment) bool
}

type NonconfidentialIARP struct {
	AssetID AssetID
}

type ConfidentialIARP struct {
	IssuanceKeys  []ecmath.Point
	TracingPoint  ecmath.Point
	IssuanceProof *OlegZKP
}

func CreateNonconfidentialIARP(assetID AssetID) *NonconfidentialIARP {
	return &NonconfidentialIARP{AssetID: assetID}
}

func (iarp *NonconfidentialIARP) Validate(ac *AssetCommitment) bool {
	ac2 := PointPair{ecmath.Point(CreateAssetPoint(iarp.AssetID)), ecmath.ZeroPoint}
	return (*PointPair)(ac).ConstTimeEqual(&ac2)
}
