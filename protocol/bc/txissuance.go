package bc

import (
	"chain/errors"
	"chain/protocol/vm"
)

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

func (iss *Issuance) checkValid(vs *validationState) error {
	if iss.Witness.AssetDefinition.InitialBlockID != vs.blockchainID {
		return errors.WithDetailf(errWrongBlockchain, "current blockchain %x, asset defined on blockchain %x", vs.blockchainID[:], iss.Witness.AssetDefinition.InitialBlockID[:])
	}

	computedAssetID := iss.Witness.AssetDefinition.ComputeAssetID()
	if computedAssetID != iss.Body.Value.AssetID {
		return errors.WithDetailf(errMismatchedAssetID, "asset ID is %x, issuance wants %x", computedAssetID[:], iss.Body.Value.AssetID[:])
	}

	err := vm.Verify(NewTxVMContext(vs.tx, iss, iss.Witness.AssetDefinition.IssuanceProgram, iss.Witness.Arguments))
	if err != nil {
		return errors.Wrap(err, "checking issuance program")
	}

	var anchored Hash
	switch a := iss.Anchor.(type) {
	case *Nonce:
		anchored = a.Witness.AnchoredID

	case *Spend:
		anchored = a.Witness.AnchoredID

	case *Issuance:
		anchored = a.Witness.AnchoredID

	default:
		return errors.WithDetailf(errEntryType, "issuance anchor has type %T, should be nonce, spend, or issuance", iss.Anchor)
	}

	if anchored != vs.entryID {
		return errors.WithDetailf(errMismatchedReference, "issuance %x anchor is for %x", vs.entryID[:], anchored[:])
	}

	anchorVS := *vs
	anchorVS.entryID = iss.Body.AnchorID
	err = iss.Anchor.checkValid(&anchorVS)
	if err != nil {
		return errors.Wrap(err, "checking issuance anchor")
	}

	destVS := *vs
	destVS.destPos = 0
	err = iss.Witness.Destination.CheckValid(&destVS)
	if err != nil {
		return errors.Wrap(err, "checking issuance destination")
	}

	if vs.tx.Body.Version == 1 && (iss.Body.ExtHash != Hash{}) {
		return errNonemptyExtHash
	}

	return nil
}
