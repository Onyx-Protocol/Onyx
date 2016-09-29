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
func ComputeAssetID(issuanceProgram []byte, initialHash [32]byte, vmVersion uint32) (assetID AssetID) {
	h := sha3pool.Get()
	defer sha3pool.Put(h)
	h.Write(initialHash[:])
	blockchain.WriteUvarint(h, uint64(assetVersion))
	blockchain.WriteUvarint(h, uint64(vmVersion))
	blockchain.WriteBytes(h, issuanceProgram)
	h.Read(assetID[:])
	return assetID
}
