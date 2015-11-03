package bc

import (
	"database/sql/driver"

	"chain/crypto/hash256"
	"chain/fedchain/script"
)

type (
	// AssetID is the Hash160 of the issuance script
	// for the asset and the genesis block of the chain
	// where it appears.
	AssetID [32]byte
)

func (a AssetID) String() string                { return Hash(a).String() }
func (a AssetID) MarshalText() ([]byte, error)  { return Hash(a).MarshalText() }
func (a *AssetID) UnmarshalText(b []byte) error { return (*Hash)(a).UnmarshalText(b) }
func (a AssetID) Value() (driver.Value, error)  { return Hash(a).Value() }
func (a *AssetID) Scan(b interface{}) error     { return (*Hash)(a).Scan(b) }

// ComputeAssetID computes the asset ID of the asset defined by
// the given issuance script and genesis block hash.
func ComputeAssetID(issuance script.Script, genesis [32]byte) AssetID {
	buf := append([]byte{}, genesis[:]...)
	sh := hash256.Sum(issuance)
	buf = append(buf, sh[:]...)
	return hash256.Sum(buf)
}

// AssetDefinitionPointer is a Hash256 value of data associated
// with a specific AssetID.
// This is issuer's authenticated description of their asset.
type AssetDefinitionPointer struct {
	AssetID        AssetID
	DefinitionHash [32]byte
}
