package tx

import "chain/protocol/bc"

type issuance struct {
	body struct {
		Anchor  bc.Hash
		Value   bc.AssetAmount
		Data    bc.Hash
		ExtHash bc.Hash
	}
	ordinal int

	// Anchor is a pointer to the manifested entry corresponding to
	// body.Anchor.
	Anchor entry
}

func (issuance) Type() string           { return "issuance1" }
func (iss *issuance) Body() interface{} { return iss.body }

func (iss issuance) Ordinal() int { return iss.ordinal }

func newIssuance(anchor entry, value bc.AssetAmount, data bc.Hash, ordinal int) *issuance {
	iss := new(issuance)
	if anchor != nil {
		iss.body.Anchor = entryID(anchor)
		iss.Anchor = anchor
	}
	iss.body.Value = value
	iss.body.Data = data
	iss.ordinal = ordinal
	return iss
}
