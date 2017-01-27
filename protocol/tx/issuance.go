package tx

import "chain/protocol/bc"

type issuance struct {
	Anchor  entryRef
	Value   bc.AssetAmount
	Data    entryRef
	ExtHash extHash
}

func (issuance) Type() string { return "issuance1" }

func newIssuance(anchor entryRef, value bc.AssetAmount, data entryRef) *entry {
	return &entry{
		body: &issuance{
			Anchor: anchor,
			Value:  value,
			Data:   data,
		},
	}
}
