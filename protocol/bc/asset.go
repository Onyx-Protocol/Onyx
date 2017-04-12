package bc

import (
	"database/sql/driver"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
)

// AssetID is the Hash256 of the asset definition.

func NewAssetID(b [32]byte) (a AssetID) {
	return AssetID(NewHash(b))
}

func (a AssetID) Byte32() (b32 [32]byte)               { return Hash(a).Byte32() }
func (a AssetID) MarshalText() ([]byte, error)         { return Hash(a).MarshalText() }
func (a *AssetID) UnmarshalText(b []byte) error        { return (*Hash)(a).UnmarshalText(b) }
func (a *AssetID) UnmarshalJSON(b []byte) error        { return (*Hash)(a).UnmarshalJSON(b) }
func (a AssetID) Bytes() []byte                        { return Hash(a).Bytes() }
func (a AssetID) Value() (driver.Value, error)         { return Hash(a).Value() }
func (a *AssetID) Scan(val interface{}) error          { return (*Hash)(a).Scan(val) }
func (a AssetID) WriteTo(w io.Writer) (int64, error)   { return Hash(a).WriteTo(w) }
func (a *AssetID) ReadFrom(r io.Reader) (int64, error) { return (*Hash)(a).ReadFrom(r) }
func (a *AssetID) IsZero() bool                        { return (*Hash)(a).IsZero() }

func (ad *AssetDefinition) ComputeAssetID() (assetID AssetID) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	writeForHash(h, *ad) // error is impossible
	assetID.ReadFrom(h)
	return assetID
}

func ComputeAssetID(prog []byte, initialBlockID *Hash, vmVersion uint64, data *Hash) AssetID {
	def := &AssetDefinition{
		InitialBlockId: initialBlockID,
		IssuanceProgram: &Program{
			VmVersion: vmVersion,
			Code:      prog,
		},
		Data: data,
	}
	return def.ComputeAssetID()
}

func (a *AssetAmount) ReadFrom(r io.Reader) (int64, error) {
	var assetID AssetID
	n, err := assetID.ReadFrom(r)
	if err != nil {
		return n, err
	}
	a.AssetId = &assetID
	var n2 int
	a.Amount, n2, err = blockchain.ReadVarint63(r)
	return n + int64(n2), err
}

func (a AssetAmount) WriteTo(w io.Writer) (int64, error) {
	n, err := a.AssetId.WriteTo(w)
	if err != nil {
		return n, err
	}
	n2, err := blockchain.WriteVarint63(w, a.Amount)
	return n + int64(n2), err
}

func (a AssetAmount) Equal(other *AssetAmount) bool {
	return a.Amount == other.Amount && *a.AssetId == *other.AssetId
}
