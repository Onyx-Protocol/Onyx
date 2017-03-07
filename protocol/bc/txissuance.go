package bc

type Issuance struct {
	body struct {
		Anchor  Hash
		Value   AssetAmount
		Data    Hash
		ExtHash Hash
	}
	ordinal int

	// Anchor is a pointer to the manifested entry corresponding to
	// body.Anchor.
	Anchor Entry // *nonce or *spend
}

func (Issuance) Type() string           { return "issuance1" }
func (iss *Issuance) Body() interface{} { return iss.body }

func (iss Issuance) Ordinal() int { return iss.ordinal }

func newIssuance(anchor Entry, value AssetAmount, data Hash, ordinal int) *Issuance {
	iss := new(Issuance)
	iss.body.Anchor = EntryID(anchor)
	iss.Anchor = anchor
	iss.body.Value = value
	iss.body.Data = data
	iss.ordinal = ordinal
	return iss
}
