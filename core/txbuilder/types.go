package txbuilder

import (
	"context"
	"encoding/json"
	"time"

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

	// Final signals to Sign that signatures should commit to the tx
	// sighash rather than to constraints based on individual inputs and
	// outputs. This effectively prevents further changes to the tx.
	Final bool `json:"final"`

	sigHasher *bc.SigHasher
}

func (t *Template) Hash(idx int) bc.Hash {
	if t.sigHasher == nil {
		t.sigHasher = bc.NewSigHasher(t.Transaction)
	}
	return t.sigHasher.Hash(idx)
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
		Position          uint32            `json:"position"`
		WitnessComponents []json.RawMessage `json:"witness_components"`
	}
	err := json.Unmarshal(b, &pre)
	if err != nil {
		return err
	}
	si.AssetAmount = pre.AssetAmount
	si.Position = pre.Position

	for i, w := range pre.WitnessComponents {
		var t struct {
			Type string `json:"type"`
		}
		err = json.Unmarshal(w, &t)
		if err != nil {
			return err
		}
		var component WitnessComponent
		switch t.Type {
		case "signature":
			var s SignatureWitness
			err = json.Unmarshal(w, &s)
			if err != nil {
				return err
			}
			component = &s
		default:
			return errors.WithDetailf(ErrBadWitnessComponent, "witness component %d has unknown type '%s'", i, t.Type)
		}
		si.WitnessComponents = append(si.WitnessComponents, component)
	}
	return nil
}

type Action interface {
	Build(context.Context, time.Time) ([]*bc.TxInput, []*bc.TxOutput, []*SigningInstruction, error)
}

type ttler interface {
	GetTTL() time.Duration
}
