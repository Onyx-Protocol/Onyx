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

func (a AssetID) String() string               { h := NewHash(a); return h.String() }
func (a AssetID) MarshalText() ([]byte, error) { return NewHash(a).MarshalText() }
func (a AssetID) Value() (driver.Value, error) { return NewHash(a).Value() }
func (a *AssetID) UnmarshalText(b []byte) error {
	var h Hash
	err := h.UnmarshalText(b)
	*a = h.Byte32()
	return err
}

func (a *AssetID) UnmarshalJSON(b []byte) error {
	var h Hash
	err := h.UnmarshalJSON(b)
	*a = h.Byte32()
	return err
}
func (a *AssetID) Scan(v interface{}) error {
	var h Hash
	err := h.Scan(v)
	*a = h.Byte32()
	return err
}

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

type AssetAmount struct {
	AssetID AssetID `json:"asset_id"`
	Amount  uint64  `json:"amount"`
}

func (a *AssetAmount) readFrom(r io.Reader) (int, error) {
	n1, err := io.ReadFull(r, a.AssetID[:])
	if err != nil {
		return int(n1), err
	}
	var n2 int
	a.Amount, n2, err = blockchain.ReadVarint63(r)
	return int(n1) + n2, err
}

func (a *AssetAmount) writeTo(w io.Writer) error {
	_, err := w.Write(a.AssetID[:])
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, a.Amount)
	return err
}
