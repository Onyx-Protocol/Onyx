package bc

import (
	"fmt"

	"chain/errors"
	"chain/protocol/bc"
)

// Issuance is a source of new value on a blockchain. It satisfies the
// Entry interface.
//
// (Not to be confused with the deprecated type IssuanceInput.)
type Issuance struct {
	body struct {
		Anchor  Hash
		Value   AssetAmount
		Data    Hash
		ExtHash Hash
	}
	ordinal int

	witness struct {
		Destination     ValueDestination
		AssetDefinition AssetDefinition
		Arguments       [][]byte
		Anchored        Hash
	}

	// Anchor is a pointer to the manifested entry corresponding to
	// body.Anchor.
	Anchor Entry // *nonce, *spend, or *issuance

	// Anchored is a pointer to the manifested entry corresponding to
	// witness.Anchored.
	Anchored Entry
}

func (Issuance) Type() string           { return "issuance1" }
func (iss *Issuance) Body() interface{} { return iss.body }

func (iss Issuance) Ordinal() int { return iss.ordinal }

func (iss *Issuance) AnchorID() Hash {
	return iss.body.Anchor
}

func (iss *Issuance) Data() Hash {
	return iss.body.Data
}

func (iss *Issuance) AssetID() AssetID {
	return iss.body.Value.AssetID
}

func (iss *Issuance) Amount() uint64 {
	return iss.body.Value.Amount
}

func (iss *Issuance) Destination() ValueDestination {
	return iss.witness.Destination
}

func (iss *Issuance) InitialBlockID() Hash {
	return iss.witness.InitialBlockID
}

func (iss *Issuance) IssuanceProgram() Program {
	return iss.witness.IssuanceProgram
}

func (iss *Issuance) Arguments() [][]byte {
	return iss.witness.Arguments
}

func (iss *Issuance) SetDestination(id Hash, pos uint64, e Entry) {
	iss.witness.Destination = ValueDestination{
		Ref:      id,
		Position: pos,
		Entry:    e,
	}
}

func (iss *Issuance) SetInitialBlockID(hash Hash) {
	iss.witness.InitialBlockID = hash
}

func (iss *Issuance) SetAssetDefinitionHash(hash Hash) {
	iss.witness.AssetDefinitionHash = hash
}

func (iss *Issuance) SetIssuanceProgram(prog Program) {
	iss.witness.IssuanceProgram = prog
}

func (iss *Issuance) SetArguments(args [][]byte) {
	iss.witness.Arguments = args
}

// NewIssuance creates a new Issuance.
func NewIssuance(anchor Entry, value AssetAmount, data Hash, ordinal int) *Issuance {
	iss := new(Issuance)
	iss.body.Anchor = EntryID(anchor)
	iss.Anchor = anchor
	iss.body.Value = value
	iss.body.Data = data
	iss.ordinal = ordinal
	return iss
}

func (iss *Issuance) CheckValid(state *validationState) error {
	if iss.witness.AssetDefinition.InitialBlockID != state.initialBlockID {
		// xxx error
	}

	if iss.witness.AssetDefinition.ComputeAssetID() != iss.body.Value.AssetID {
		// xxx error
	}

	// xxx run issuance program

	id := EntryID(iss) // xxx can this be supplied from somewhere?

	switch a := iss.Anchor.(type) {
	case *Nonce:
		if a.witness.Anchored != id {
			// xxx error
		}

	case *Spend:
		if a.witness.Anchored != id {
			// xxx error
		}

	case *Issuance:
		if a.witness.Anchored != id {
			// xxx error
		}

	default:
		return fmt.Errorf("issuance anchor has type %T, should be nonce, spend, or issuance", iss.Anchor)
	}

	anchorState := *state
	anchorState.currentEntryID = id
	err := iss.Anchor.CheckValid(&anchorState)
	if err != nil {
		return errors.Wrap(err, "checking issuance anchor")
	}

	destState := *state
	destState.currentEntryID = id
	destState.destPosition = 0
	err = iss.witness.Destination.CheckValid(&destState)
	if err != nil {
		return errors.Wrap(err, "checking issuance destination")
	}

	if state.txVersion == 1 && (iss.body.ExtHash != bc.Hash{}) {
		// xxx error
	}

	return nil
}
