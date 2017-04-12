package txbuilder

import (
	"bytes"
	"math"
	"time"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
)

func NewBuilder(maxTime time.Time) *TemplateBuilder {
	return &TemplateBuilder{maxTime: maxTime}
}

type TemplateBuilder struct {
	base                *legacy.TxData
	inputs              []*legacy.TxInput
	outputs             []*legacy.TxOutput
	signingInstructions []*SigningInstruction
	minTime             time.Time
	maxTime             time.Time
	referenceData       []byte
	rollbacks           []func()
	callbacks           []func() error
}

func (b *TemplateBuilder) AddInput(in *legacy.TxInput, sigInstruction *SigningInstruction) error {
	if in.Amount() > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", in.Amount())
	}
	b.inputs = append(b.inputs, in)
	b.signingInstructions = append(b.signingInstructions, sigInstruction)
	return nil
}

func (b *TemplateBuilder) AddOutput(o *legacy.TxOutput) error {
	if o.Amount > math.MaxInt64 {
		return errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", o.Amount)
	}
	b.outputs = append(b.outputs, o)
	return nil
}

func (b *TemplateBuilder) RestrictMinTime(t time.Time) {
	if t.After(b.minTime) {
		b.minTime = t
	}
}

func (b *TemplateBuilder) RestrictMaxTime(t time.Time) {
	if t.Before(b.maxTime) {
		b.maxTime = t
	}
}

func (b *TemplateBuilder) MaxTime() time.Time {
	return b.maxTime
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

func (b *TemplateBuilder) Build() (*Template, *legacy.TxData, error) {
	// Run any building callbacks.
	for _, cb := range b.callbacks {
		err := cb()
		if err != nil {
			return nil, nil, err
		}
	}

	tpl := &Template{}
	tx := b.base
	if tx == nil {
		tx = &legacy.TxData{
			Version: 1,
		}
		tpl.Local = true
	}

	// Update min & max times.
	if !b.minTime.IsZero() && bc.Millis(b.minTime) > tx.MinTime {
		tx.MinTime = bc.Millis(b.minTime)
	}
	if tx.MaxTime == 0 || tx.MaxTime > bc.Millis(b.maxTime) {
		tx.MaxTime = bc.Millis(b.maxTime)
	}

	// Set transaction reference data if applicable.
	if len(b.referenceData) > 0 {
		tx.ReferenceData = b.referenceData
	}

	// Add all the built outputs.
	tx.Outputs = append(tx.Outputs, b.outputs...)

	// Add all the built inputs and their corresponding signing instructions.
	for i, in := range b.inputs {
		instruction := b.signingInstructions[i]
		instruction.Position = uint32(len(tx.Inputs))

		// Empty signature arrays should be serialized as empty arrays, not null.
		if instruction.SignatureWitnesses == nil {
			instruction.SignatureWitnesses = []*signatureWitness{}
		}
		tpl.SigningInstructions = append(tpl.SigningInstructions, instruction)
		tx.Inputs = append(tx.Inputs, in)
	}
	tpl.Transaction = legacy.NewTx(*tx)
	return tpl, tx, nil
}
