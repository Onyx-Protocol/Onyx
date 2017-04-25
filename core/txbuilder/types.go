package txbuilder

import (
	"context"
	"encoding/json"
	"time"

	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
)

// Template represents a partially- or fully-signed transaction.
type Template struct {
	Transaction         *legacy.Tx            `json:"raw_transaction"`
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
}

func (t *Template) Hash(idx uint32) bc.Hash {
	return t.Transaction.SigHash(idx)
}

// SigningInstruction gives directions for signing inputs in a TxTemplate.
type SigningInstruction struct {
	Position           uint32              `json:"position"`
	SignatureWitnesses []*signatureWitness `json:"witness_components,omitempty"`
}

func (si *SigningInstruction) UnmarshalJSON(b []byte) error {
	var pre struct {
		Position           uint32 `json:"position"`
		SignatureWitnesses []struct {
			Type string
			signatureWitness
		} `json:"witness_components"`
	}
	err := json.Unmarshal(b, &pre)
	if err != nil {
		return err
	}

	si.Position = pre.Position
	si.SignatureWitnesses = make([]*signatureWitness, 0, len(pre.SignatureWitnesses))
	for i, w := range pre.SignatureWitnesses {
		if w.Type != "signature" {
			return errors.WithDetailf(ErrBadWitnessComponent, "witness component %d has unknown type '%s'", i, w.Type)
		}
		si.SignatureWitnesses = append(si.SignatureWitnesses, &w.signatureWitness)
	}
	return nil
}

type Action interface {
	Build(context.Context, *TemplateBuilder) error
}

// Receiver encapsulates information about where to send assets.
type Receiver struct {
	ControlProgram chainjson.HexBytes `json:"control_program"`
	ExpiresAt      time.Time          `json:"expires_at"`
}
