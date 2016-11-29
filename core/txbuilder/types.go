package txbuilder

import (
	"context"
	"encoding/json"
	"time"

	"chain-stealth/crypto/ca"
	chainjson "chain-stealth/encoding/json"
	"chain-stealth/errors"
	"chain-stealth/protocol/bc"
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

	ConfidentialityInstructions []*ConfidentialityInstruction `json:"confidentiality_instructions"`

	sigHasher *bc.SigHasher
}

type ConfidentialityInstruction struct {
	Type                     string             `json:"type"` // 'input' or 'output'
	Value                    uint64             `json:"value"`
	AssetID                  bc.AssetID         `json:"asset_id"`
	AssetCommitment          ca.AssetCommitment `json:"asset_commitment"`
	ValueBlindingFactor      ca.Scalar          `json:"value_blinding_factor"`
	CumulativeBlindingFactor ca.Scalar          `json:"cumulative_blinding_factor"`
}

func (ci *ConfidentialityInstruction) IsInput() bool {
	return ci.Type == "input"
}

func (ci *ConfidentialityInstruction) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":                       ci.Type,
		"value":                      ci.Value,
		"asset_id":                   ci.AssetID.String(),
		"asset_commitment":           chainjson.HexBytes(ci.AssetCommitment.Bytes()),
		"value_blinding_factor":      chainjson.HexBytes(ci.ValueBlindingFactor[:]),
		"cumulative_blinding_factor": chainjson.HexBytes(ci.CumulativeBlindingFactor[:]),
	})
}

func (ci *ConfidentialityInstruction) UnmarshalJSON(b []byte) error {
	var x struct {
		Type                     string             `json:"type"`
		Value                    uint64             `json:"value"`
		AssetID                  bc.AssetID         `json:"asset_id"`
		AssetCommitment          chainjson.HexBytes `json:"asset_commitment"`
		ValueBlindingFactor      chainjson.HexBytes `json:"value_blinding_factor"`
		CumulativeBlindingFactor chainjson.HexBytes `json:"cumulative_blinding_factor"`
	}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}
	var ac [32]byte
	copy(ac[:], x.AssetCommitment)
	err = ci.AssetCommitment.FromBytes(&ac)
	if err != nil {
		return err
	}
	ci.Type = x.Type
	ci.Value = x.Value
	ci.AssetID = x.AssetID
	copy(ci.ValueBlindingFactor[:], x.ValueBlindingFactor)
	copy(ci.CumulativeBlindingFactor[:], x.CumulativeBlindingFactor)
	return nil
}

func (t *Template) Hash(idx int) bc.Hash {
	if t.sigHasher == nil {
		t.sigHasher = bc.NewSigHasher(t.Transaction)
	}
	return t.sigHasher.Hash(idx)
}

// SigningInstruction gives directions for signing inputs in a TxTemplate.
type SigningInstruction struct {
	Position int `json:"position"`
	bc.AssetAmount
	WitnessComponents []WitnessComponent `json:"witness_components,omitempty"`
}

func (si *SigningInstruction) UnmarshalJSON(b []byte) error {
	var pre struct {
		bc.AssetAmount
		Position          int `json:"position"`
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
	// TODO(bobg, jeffomatic): see if there is a way to remove the maxTime
	// parameter from the build call. One possibility would be to treat TTL as
	// a transaction-wide default parameter that gets folded into actions that
	// care about it. This could happen when the build request is being
	// deserialized.
	Build(context.Context, time.Time, *TemplateBuilder) error
}
