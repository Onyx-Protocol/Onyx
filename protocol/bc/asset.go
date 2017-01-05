package bc

import (
	"database/sql/driver"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
)

const assetVersion = 1

// AssetID is the Hash256 of the issuance script for the asset and the
// initial block of the chain where it appears.
type AssetID [32]byte

func (a AssetID) String() string                { return Hash(a).String() }
func (a AssetID) MarshalText() ([]byte, error)  { return Hash(a).MarshalText() }
func (a *AssetID) UnmarshalText(b []byte) error { return (*Hash)(a).UnmarshalText(b) }
func (a *AssetID) UnmarshalJSON(b []byte) error { return (*Hash)(a).UnmarshalJSON(b) }
func (a AssetID) Value() (driver.Value, error)  { return Hash(a).Value() }
func (a *AssetID) Scan(b interface{}) error     { return (*Hash)(a).Scan(b) }

// ComputeAssetID computes the asset ID of the asset defined by
// the given issuance program and initial block hash.
func ComputeAssetID(issuanceProgram []byte, initialHash [32]byte, vmVersion uint64, assetDefinitionHash Hash) (assetID AssetID) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(initialHash[:])
	blockchain.WriteVarint63(h, vmVersion)
	blockchain.WriteVarstr31(h, issuanceProgram) // TODO(bobg): check and return error
	if assetDefinitionHash == EmptyHash {
		blockchain.WriteVarstr31(h, nil)
	} else {
		blockchain.WriteVarstr31(h, assetDefinitionHash[:])
	}
	h.Read(assetID[:])
	return assetID
}
