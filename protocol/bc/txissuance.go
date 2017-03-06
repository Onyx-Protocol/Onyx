package bc

type issuance struct {
	body struct {
		Anchor  Hash
		Value   AssetAmount
		Data    Hash
		ExtHash Hash
	}
	ordinal int

	// Anchor is a pointer to the manifested entry corresponding to
	// body.Anchor.
	Anchor entry // *nonce or *spend
}

func (issuance) Type() string           { return "issuance1" }
func (iss *issuance) Body() interface{} { return iss.body }

func (iss issuance) Ordinal() int { return iss.ordinal }

func newIssuance(anchor entry, value AssetAmount, data Hash, ordinal int) *issuance {
	iss := new(issuance)
	iss.body.Anchor = entryID(anchor)
	iss.Anchor = anchor
	iss.body.Value = value
	iss.body.Data = data
	iss.ordinal = ordinal
	return iss
}
