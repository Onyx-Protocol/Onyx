package bc

import (
	"database/sql/driver"

	"golang.org/x/crypto/sha3"
)

// AssetID is the Hash256 of the issuance script for the asset and the
// genesis block of the chain where it appears.
type AssetID [32]byte

func (a AssetID) String() string                { return Hash(a).String() }
func (a AssetID) MarshalText() ([]byte, error)  { return Hash(a).MarshalText() }
func (a *AssetID) UnmarshalText(b []byte) error { return (*Hash)(a).UnmarshalText(b) }
func (a AssetID) Value() (driver.Value, error)  { return Hash(a).Value() }
func (a *AssetID) Scan(b interface{}) error     { return (*Hash)(a).Scan(b) }

// ComputeAssetID computes the asset ID of the asset defined by
// the given issuance script and genesis block hash.
func ComputeAssetID(issuanceScript []byte, genesis [32]byte) AssetID {
	buf := append([]byte{}, genesis[:]...)
	sh := sha3.Sum256(issuanceScript)
	buf = append(buf, sh[:]...)
	return sha3.Sum256(buf)
}

// HashAssetDefinition calculates an asset definition's hash.
func HashAssetDefinition(def []byte) Hash {
	return Hash(sha3.Sum256(def))
}
