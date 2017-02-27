package tx

import "chain/protocol/bc"

type issuance struct {
	body struct {
		Anchor  *EntryRef
		Value   bc.AssetAmount
		Data    bc.Hash
		ExtHash bc.Hash
	}
	ordinal int
}

func (issuance) Type() string           { return "issuance1" }
func (iss *issuance) Body() interface{} { return iss.body }

func (iss issuance) Ordinal() int { return iss.ordinal }

func newIssuance(anchor *EntryRef, value bc.AssetAmount, data bc.Hash, ordinal int) *issuance {
	iss := new(issuance)
	iss.body.Anchor = anchor
	iss.body.Value = value
	iss.body.Data = data
	iss.ordinal = ordinal
	return iss
}
