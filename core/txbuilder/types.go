package txbuilder

import (
	"context"
	"encoding/json"

	"chain/errors"
	"chain/protocol/bc"
)

// Template represents a partially- or fully-signed transaction.
type Template struct {
	Transaction         *bc.TxData            `json:"raw_transaction"`
	SigningInstructions []*SigningInstruction `json:"signing_instructions"`

	// Local indicates that all inputs to the transaction are signed
	// exclusively by keys managed by this Core. Whenever accepting
	// a template from an external Core, `Local` should be set to
	// false.
	Local bool `json:"local"`

	// AllowAdditional affects whether Sign commits to the tx sighash or
	// to individual details of the tx so far. When true, signatures
	// commit to tx details, and new details may be added but existing
	// ones cannot be changed. When false, signatures commit to the tx
	// as a whole, and any change to the tx invalidates the signature.
	AllowAdditional bool `json:"allow_additional_actions"`

	sigHasher *bc.SigHasher
}

func (t *Template) Hash(idx uint32) bc.Hash {
	if t.sigHasher == nil {
		t.sigHasher = bc.NewSigHasher(t.Transaction)
	}
	return t.sigHasher.Hash(int(idx))
}

// SigningInstruction gives directions for signing inputs in a TxTemplate.
type SigningInstruction struct {
	Position uint32 `json:"position"`
	bc.AssetAmount
	WitnessComponents []WitnessComponent `json:"witness_components,omitempty"`
}

func (si *SigningInstruction) UnmarshalJSON(b []byte) error {
	var pre struct {
		bc.AssetAmount
		Position          uint32 `json:"position"`
		WitnessComponents []struct {
			Type string
			SignatureWitness
		} `json:"witness_components"`
	}
	err := json.Unmarshal(b, &pre)
	if err != nil {
		return err
	}

	si.AssetAmount = pre.AssetAmount
	si.Position = pre.Position
	si.WitnessComponents = make([]WitnessComponent, 0, len(pre.WitnessComponents))
	for i, w := range pre.WitnessComponents {
		if w.Type != "signature" {
			return errors.WithDetailf(ErrBadWitnessComponent, "witness component %d has unknown type '%s'", i, w.Type)
		}
		si.WitnessComponents = append(si.WitnessComponents, &w.SignatureWitness)
	}
	return nil
}

type Action interface {
	Build(context.Context, *TemplateBuilder) error
}
