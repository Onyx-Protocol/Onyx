package txbuilder

import (
	"bytes"
	"math"
	"time"

	"chain/errors"
	"chain/protocol/bc"
)

type TemplateBuilder struct {
	base                *bc.TxData
	maxTime             time.Time
	inputs              []*bc.TxInput
	outputs             []*bc.TxOutput
	signingInstructions []*SigningInstruction
	minTimeMS           uint64
	referenceData       []byte
	rollbacks           []func()
	callbacks           []func() error
	values              map[interface{}]interface{}
}

func (b *TemplateBuilder) AddInput(in *bc.TxInput, sigInstruction *SigningInstruction) error {
	if in.Amount() > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", in.Amount())
	}
	b.inputs = append(b.inputs, in)
	b.signingInstructions = append(b.signingInstructions, sigInstruction)
	return nil
}

func (b *TemplateBuilder) AddOutput(o *bc.TxOutput) error {
	if o.Amount > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", o.Amount)
	}
	b.outputs = append(b.outputs, o)
	return nil
}

func (b *TemplateBuilder) RestrictMinTimeMS(ms uint64) {
	if ms > b.minTimeMS {
		b.minTimeMS = ms
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
	if tpl.Transaction.MaxTime == 0 || tpl.Transaction.MaxTime > bc.Millis(b.maxTime) {
		tpl.Transaction.MaxTime = bc.Millis(b.maxTime)
	}

	// Set transaction reference data if applicable.
	if len(b.referenceData) > 0 {
		tpl.Transaction.ReferenceData = b.referenceData
	}

	// Add all the built outputs.
	tpl.Transaction.Outputs = append(tpl.Transaction.Outputs, b.outputs...)

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
	return tpl, nil
}
