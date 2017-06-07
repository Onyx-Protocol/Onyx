package ca

// AssetID is a 32-byte unique identifier of an asset type.
// We are not using blockchain type AssetID to avoid a circular dependency.
type AssetID [32]byte
