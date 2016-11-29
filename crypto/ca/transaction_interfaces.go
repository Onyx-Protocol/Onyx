// This provides interfaces for integrating with blockchain data structures.

package ca

type Issuance interface {
	AssetDescriptor() AssetDescriptor
	ValueDescriptor() ValueDescriptor
	AssetIDs() []AssetID
	IssuanceAssetRangeProof() *IssuanceAssetRangeProof
	ValueRangeProof() *ValueRangeProof
}

type Spend interface {
	AssetDescriptor() AssetDescriptor
	ValueDescriptor() ValueDescriptor
}

type Output interface {
	AssetDescriptor() AssetDescriptor
	ValueDescriptor() ValueDescriptor
	AssetRangeProof() *AssetRangeProof
	ValueRangeProof() *ValueRangeProof
}
