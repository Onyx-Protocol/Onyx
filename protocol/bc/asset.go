package bc

import (
	"database/sql/driver"
	"io"

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
	h.Write(assetDefinitionHash[:])
	h.Read(assetID[:])
	return assetID
}

type AssetAmount struct {
	AssetID AssetID `json:"asset_id"`
	Amount  uint64  `json:"amount"`
}

// assumes r has sticky errors
func (a *AssetAmount) readFrom(r io.Reader) (int, error) {
	n1, err := io.ReadFull(r, a.AssetID[:])
	if err != nil {
		return n1, err
	}
	var n2 int
	a.Amount, n2, err = blockchain.ReadVarint63(r)
	return n1 + n2, err
}

func (a *AssetAmount) writeTo(w io.Writer) error {
	_, err := w.Write(a.AssetID[:])
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, a.Amount)
	return err
}
