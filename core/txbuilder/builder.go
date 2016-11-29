package txbuilder

import (
	"bytes"
	"fmt"
	"math"

	"chain-stealth/crypto/ca"
	"chain-stealth/errors"
	"chain-stealth/protocol/bc"
)

type TemplateBuilder struct {
	base                *bc.TxData
	inputs              []*bc.TxInput
	signingInstructions []*SigningInstruction // parallel inputs
	signedInputs        []*bc.TxInput
	rawOutputs          []*bc.TxOutput
	encryptedOutputs    []*bc.TxOutput
	outputREKs          []ca.RecordKey // parallel to encryptedOutputs
	confInstructions    []*ConfidentialityInstruction
	minTimeMS           uint64
	maxTimeMS           uint64
	referenceData       []byte
	rollbacks           []func()
	callbacks           []func() error
}

func NewTemplateBuilder() *TemplateBuilder {
	return &TemplateBuilder{} // TODO: delete this func
}

func (b *TemplateBuilder) AddRawInput(in *bc.TxInput, sigInstruction *SigningInstruction) error {
	if amt, ok := in.Amount(); ok && amt > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", amt)
	}
	b.inputs = append(b.inputs, in)
	b.signingInstructions = append(b.signingInstructions, sigInstruction)
	return nil
}

func (b *TemplateBuilder) AddInput(in *bc.TxInput, aa bc.AssetAmount, sigInstruction *SigningInstruction, v *bc.CAValues) error {
	if aa.Amount > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", aa.Amount)
	}

	b.inputs = append(b.inputs, in)
	b.confInstructions = append(b.confInstructions, &ConfidentialityInstruction{
		Type:                     "input",
		Value:                    aa.Amount,
		AssetID:                  aa.AssetID,
		AssetCommitment:          v.AssetCommitment,
		ValueBlindingFactor:      v.ValueBlindingFactor,
		CumulativeBlindingFactor: v.CumulativeBlindingFactor,
	})
	b.signingInstructions = append(b.signingInstructions, sigInstruction)
	return nil
}

func (b *TemplateBuilder) AddRawOutput(o *bc.TxOutput) error {
	if amt, ok := o.Amount(); ok && amt > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", amt)
	}

	b.rawOutputs = append(b.rawOutputs, o)
	return nil
}

func (b *TemplateBuilder) AddEncryptedOutput(o *bc.TxOutput, rek ca.RecordKey) error {
	aa, ok := o.GetAssetAmount()
	if !ok {
		return errors.New("provided output is already encrypted")
	}
	if aa.Amount > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", aa.Amount)
	}

	b.encryptedOutputs = append(b.encryptedOutputs, o)
	b.outputREKs = append(b.outputREKs, rek)
	return nil
}

func (b *TemplateBuilder) RestrictMinTimeMS(ms uint64) {
	if ms > b.minTimeMS {
		b.minTimeMS = ms
	}
}

func (b *TemplateBuilder) RestrictMaxTimeMS(ms uint64) {
	if ms < b.maxTimeMS || b.maxTimeMS == 0 {
		b.maxTimeMS = ms
	}
}

// OnRollback registers a function that can be
// used to attempt to undo any side effects of building
// actions. For example, it might cancel any reservations
// reservations that were made on UTXOs in a spend action.
// Rollback is a "best-effort" operation and not guaranteed
// to succeed. Each action's side effects, if any, must be
// designed with this in mind.
func (b *TemplateBuilder) OnRollback(rollbackFn func()) {
	b.rollbacks = append(b.rollbacks, rollbackFn)
}

// OnBuild registers a function that will be run after all
// actions have been successfully built.
func (b *TemplateBuilder) OnBuild(buildFn func() error) {
	b.callbacks = append(b.callbacks, buildFn)
}

func (b *TemplateBuilder) setReferenceData(data []byte) error {
	if b.base != nil && len(b.base.ReferenceData) != 0 && !bytes.Equal(b.base.ReferenceData, data) {
		return errors.Wrap(ErrBadRefData)
	}
	if len(b.referenceData) != 0 && !bytes.Equal(b.referenceData, data) {
		return errors.Wrap(ErrBadRefData)
	}
	b.referenceData = data
	return nil
}

func (b *TemplateBuilder) rollback() {
	for _, f := range b.rollbacks {
		f()
	}
}

func (b *TemplateBuilder) Build() (*Template, error) {
	// Run any building callbacks.
	for _, cb := range b.callbacks {
		err := cb()
		if err != nil {
			return nil, err
		}
	}

	tpl := &Template{Transaction: b.base}
	if tpl.Transaction == nil {
		tpl.Transaction = &bc.TxData{
			Version: bc.CurrentTransactionVersion,
		}
		tpl.Local = true
	}

	// Update min & max times.
	if b.minTimeMS > 0 && b.minTimeMS > tpl.Transaction.MinTime {
		tpl.Transaction.MinTime = b.minTimeMS
	}
	if tpl.Transaction.MaxTime == 0 || tpl.Transaction.MaxTime > b.maxTimeMS {
		tpl.Transaction.MaxTime = b.maxTimeMS
	}

	// Set transaction reference data if applicable.
	if len(b.referenceData) > 0 {
		tpl.Transaction.ReferenceData = b.referenceData
	}

	// Add all of the presigned inputs.
	tpl.Transaction.Inputs = append(tpl.Transaction.Inputs, b.signedInputs...)

	// Add all the built inputs and their corresponding signing instructions.
	for i, in := range b.inputs {
		instruction := b.signingInstructions[i]
		instruction.Position = len(tpl.Transaction.Inputs)

		// Empty signature arrays should be serialized as empty arrays, not null.
		if instruction.WitnessComponents == nil {
			instruction.WitnessComponents = []WitnessComponent{}
		}
		tpl.SigningInstructions = append(tpl.SigningInstructions, instruction)
		tpl.Transaction.Inputs = append(tpl.Transaction.Inputs, in)
	}

	// Add all of the raw outputs first.
	tpl.Transaction.Outputs = append(tpl.Transaction.Outputs, b.rawOutputs...)

	// Organize confidentiality instructions into a map, keyed by asset ID
	// and a slice of asset commitments.
	seenCommitments := make(map[ca.AssetCommitment]bool)
	assetCommitments := make([]ca.AssetCommitment, 0, len(b.confInstructions))
	confInstructions := make(map[bc.AssetID]*ConfidentialityInstruction)
	for _, ci := range b.confInstructions {
		if !ci.IsInput() || seenCommitments[ci.AssetCommitment] {
			continue
		}
		confInstructions[ci.AssetID] = ci
		seenCommitments[ci.AssetCommitment] = true
		assetCommitments = append(assetCommitments, ci.AssetCommitment)
	}

	// Add all of the encrypted outputs.
	if len(b.encryptedOutputs) > 0 {
		// Encrypt all but one of the outputs, saving the last one for
		// the excess factor.
		for i, o := range b.encryptedOutputs[:len(b.encryptedOutputs)-1] {
			aid, _ := o.AssetID()
			ci := confInstructions[aid]
			if ci == nil {
				return nil, fmt.Errorf("missing confidentiality instructions for %s", aid)
			}
			encrypted, vals, err := bc.EncryptedOutput(o, b.outputREKs[i], assetCommitments, ci.CumulativeBlindingFactor, nil)
			if err != nil {
				return nil, err
			}

			tpl.Transaction.Outputs = append(tpl.Transaction.Outputs, encrypted)
			b.confInstructions = append(b.confInstructions, &ConfidentialityInstruction{
				Type:                     "output",
				Value:                    vals.Value,
				AssetID:                  aid,
				AssetCommitment:          vals.AssetCommitment,
				ValueBlindingFactor:      vals.ValueBlindingFactor,
				CumulativeBlindingFactor: vals.CumulativeBlindingFactor,
			})
		}

		// Balance the blinding factors.
		// Every build call that adds a new encrypted output will have
		// balanced blinding factors.
		var inputBFTuples, outputBFTuples []ca.BFTuple
		for _, ci := range b.confInstructions {
			tup := ca.BFTuple{
				Value: ci.Value,
				C:     ci.CumulativeBlindingFactor,
				F:     ci.ValueBlindingFactor,
			}
			if ci.IsInput() {
				inputBFTuples = append(inputBFTuples, tup)
			} else {
				outputBFTuples = append(outputBFTuples, tup)
			}
		}
		excess := ca.BalanceBlindingFactors(inputBFTuples, outputBFTuples)

		// Encrypt the final output using the excess blinding factor.
		lastIdx := len(b.encryptedOutputs) - 1
		lastOut, lastREK := b.encryptedOutputs[lastIdx], b.outputREKs[lastIdx]
		aid, _ := lastOut.AssetID()
		ci := confInstructions[aid]
		if ci == nil {
			return nil, fmt.Errorf("missing confidentiality instructions for %s", aid)
		}
		encrypted, vals, err := bc.EncryptedOutput(lastOut, lastREK, assetCommitments, ci.CumulativeBlindingFactor, &excess)
		if err != nil {
			return nil, errors.Wrap(err, "encrypting output")
		}
		tpl.Transaction.Outputs = append(tpl.Transaction.Outputs, encrypted)
		b.confInstructions = append(b.confInstructions, &ConfidentialityInstruction{
			Type:                     "output",
			Value:                    vals.Value,
			AssetCommitment:          vals.AssetCommitment,
			ValueBlindingFactor:      vals.ValueBlindingFactor,
			CumulativeBlindingFactor: vals.CumulativeBlindingFactor,
		})
	}

	tpl.ConfidentialityInstructions = b.confInstructions
	return tpl, nil
}
