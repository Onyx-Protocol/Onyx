package tx

import "chain/protocol/bc"

type issuance struct {
	body struct {
		Anchor  entryRef
		Value   bc.AssetAmount
		Data    entryRef
		ExtHash extHash
	}
}

func (issuance) Type() string           { return "issuance1" }
func (iss *issuance) Body() interface{} { return iss.body }

func newIssuance(anchor entryRef, value bc.AssetAmount, data entryRef) *issuance {
	iss := new(issuance)
	iss.body.Anchor = anchor
	iss.body.Value = value
	iss.body.Data = data
	return iss
}
