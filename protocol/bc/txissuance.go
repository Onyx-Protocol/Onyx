package bc

// Issuance is a source of new value on a blockchain. It satisfies the
// Entry interface.
//
// (Not to be confused with the deprecated type IssuanceInput.)
type Issuance struct {
	Body struct {
		AnchorID Hash
		Value    AssetAmount
		Data     Hash
		ExtHash  Hash
	}
	ordinal int

	Witness struct {
		Destination     ValueDestination
		AssetDefinition AssetDefinition
		Arguments       [][]byte
		AnchoredID      Hash
	}

	// Anchor is a pointer to the manifested entry corresponding to
	// Body.AnchorID.
	Anchor Entry // *nonce, *spend, or *issuance

	// Anchored is a pointer to the manifested entry corresponding to
	// witness.AnchoredID.
	Anchored Entry
}

func (Issuance) Type() string           { return "issuance1" }
func (iss *Issuance) body() interface{} { return iss.Body }

func (iss Issuance) Ordinal() int { return iss.ordinal }

func (iss *Issuance) SetDestination(id Hash, val AssetAmount, pos uint64, e Entry) {
	iss.Witness.Destination = ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
		Entry:    e,
	}
}

// NewIssuance creates a new Issuance.
func NewIssuance(anchor Entry, value AssetAmount, data Hash, ordinal int) *Issuance {
	iss := new(Issuance)
	iss.Body.AnchorID = EntryID(anchor)
	iss.Anchor = anchor
	iss.Body.Value = value
	iss.Body.Data = data
	iss.ordinal = ordinal
	return iss
}
