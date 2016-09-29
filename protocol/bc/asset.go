package bc

import (
	"database/sql/driver"
	"sync"

	"golang.org/x/crypto/sha3"

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

var assetHashPool = &sync.Pool{New: func() interface{} { return sha3.New256() }}

// ComputeAssetID computes the asset ID of the asset defined by
// the given issuance program and initial block hash.
func ComputeAssetID(issuanceProgram []byte, initialHash [32]byte, vmVersion uint32) (assetID AssetID) {
	h := assetHashPool.Get().(sha3.ShakeHash)
	h.Write(initialHash[:])
	blockchain.WriteUvarint(h, uint64(assetVersion))
	blockchain.WriteUvarint(h, uint64(vmVersion))
	blockchain.WriteBytes(h, issuanceProgram)
	h.Read(assetID[:])
	h.Reset()
	assetHashPool.Put(h)
	return assetID
}
