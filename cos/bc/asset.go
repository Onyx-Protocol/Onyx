package bc

import (
	"bytes"
	"database/sql/driver"

	"golang.org/x/crypto/sha3"

	"chain/encoding/blockchain"
)

const assetVersion = 1

// AssetID is the Hash256 of the issuance script for the asset and the
// genesis block of the chain where it appears.
type AssetID [32]byte

func (a AssetID) String() string                { return Hash(a).String() }
func (a AssetID) MarshalText() ([]byte, error)  { return Hash(a).MarshalText() }
func (a *AssetID) UnmarshalText(b []byte) error { return (*Hash)(a).UnmarshalText(b) }
func (a AssetID) Value() (driver.Value, error)  { return Hash(a).Value() }
func (a *AssetID) Scan(b interface{}) error     { return (*Hash)(a).Scan(b) }

// ComputeAssetID computes the asset ID of the asset defined by
// the given issuance program and genesis block hash.
func ComputeAssetID(issuanceProgram []byte, genesis [32]byte, vmVersion uint32) AssetID {
	buf := append([]byte{}, genesis[:]...)
	var b bytes.Buffer
	blockchain.WriteUvarint(&b, uint64(assetVersion))
	blockchain.WriteUvarint(&b, uint64(vmVersion))
	blockchain.WriteBytes(&b, issuanceProgram)
	buf = append(buf, b.Bytes()...)
	return sha3.Sum256(buf)
}
