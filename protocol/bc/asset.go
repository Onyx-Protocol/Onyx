package bc

import (
	"database/sql/driver"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
)

// AssetID is the Hash256 of the issuance script for the asset and the
// initial block of the chain where it appears.
type AssetID [32]byte

func (a AssetID) String() string                { return byte32(a).String() }
func (a AssetID) MarshalText() ([]byte, error)  { return byte32(a).MarshalText() }
func (a *AssetID) UnmarshalText(b []byte) error { return (*byte32)(a).UnmarshalText(b) }
func (a *AssetID) UnmarshalJSON(b []byte) error { return (*byte32)(a).UnmarshalJSON(b) }
func (a AssetID) Value() (driver.Value, error)  { return byte32(a).Value() }
func (a *AssetID) Scan(b interface{}) error     { return (*byte32)(a).Scan(b) }

type AssetDefinition struct {
	InitialBlockID  Hash
	IssuanceProgram Program
	Data            Hash
}

func (ad *AssetDefinition) ComputeAssetID() (assetID AssetID) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	writeForHash(h, *ad) // error is impossible
	h.Read(assetID[:])
	return assetID
}

func ComputeAssetID(prog []byte, initialBlockID Hash, vmVersion uint64, data Hash) AssetID {
	def := &AssetDefinition{
		InitialBlockID: initialBlockID,
		IssuanceProgram: Program{
			VMVersion: vmVersion,
			Code:      prog,
		},
		Data: data,
	}
	return def.ComputeAssetID()
}

func (a *AssetID) ReadFrom(r io.Reader) (int64, error) {
	return (*byte32)(a).ReadFrom(r)
}

func (a AssetID) WriteTo(w io.Writer) (int64, error) {
	return byte32(a).WriteTo(w)
}

type AssetAmount struct {
	AssetID AssetID `json:"asset_id"`
	Amount  uint64  `json:"amount"`
}

func (a *AssetAmount) readFrom(r io.Reader) (int, error) {
	n1, err := a.AssetID.ReadFrom(r)
	if err != nil {
		return int(n1), err
	}
	var n2 int
	a.Amount, n2, err = blockchain.ReadVarint63(r)
	return int(n1) + n2, err
}

func (a *AssetAmount) writeTo(w io.Writer) error {
	_, err := a.AssetID.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, a.Amount)
	return err
}
