package txbuilder

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"chain/crypto/ed25519/hd25519"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
)

var (
	ErrBadConstraint       = errors.New("invalid witness constraint")
	ErrBadRefData          = errors.New("transaction reference data does not match previous template's reference data")
	ErrBadTxInputIdx       = errors.New("unsigned tx missing input")
	ErrBadWitnessComponent = errors.New("invalid witness component")
	ErrMissingSig          = errors.New("missing signature in template")
)

// Build builds or adds on to a transaction.
// Initially, inputs are left unconsumed, and destinations unsatisfied.
// Build partners then satisfy and consume inputs and destinations.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Build(ctx context.Context, tx *bc.TxData, actions []Action, ref json.Map, maxTime time.Time) (*Template, error) {
	if tx == nil {
		tx = &bc.TxData{
			Version: bc.CurrentTransactionVersion,
		}
	}

	// If there are any actions with a TTL, restrict the transaction's MaxTime accordingly.
	now := time.Now()
	for _, a := range actions {
		if t, ok := a.(ttler); ok {
			timestamp := now.Add(t.GetTTL())
			if timestamp.Before(maxTime) {
				maxTime = timestamp
			}
		}
	}
	if tx.MaxTime == 0 || tx.MaxTime > bc.Millis(maxTime) {
		tx.MaxTime = bc.Millis(maxTime)
	}

	if len(ref) != 0 {
		if len(tx.ReferenceData) != 0 && !bytes.Equal(tx.ReferenceData, ref) {
			return nil, errors.Wrap(ErrBadRefData)
		}

		tx.ReferenceData = ref
	}

	var tplInputs []*Input
	for i, action := range actions {
		txins, txouts, inputs, err := action.Build(ctx, maxTime)
		if err != nil {
			return nil, errors.WithDetailf(err, "invalid action %d", i)
		}

		if len(txins) != len(inputs) {
			// This would only happen from a bug in our system
			return nil, errors.Wrap(fmt.Errorf("%T returned different number of transaction and template inputs", action))
		}

		for i := range txins {
			inputs[i].Position = uint32(len(tx.Inputs))
			tplInputs = append(tplInputs, inputs[i])
			tx.Inputs = append(tx.Inputs, txins[i])
		}

		tx.Outputs = append(tx.Outputs, txouts...)
	}

	for _, input := range tplInputs {
		// Empty signature arrays should be serialized as empty arrays, not null.
		if input.WitnessComponents == nil {
			input.WitnessComponents = []WitnessComponent{}
		}
	}

	tpl := &Template{
		Unsigned: tx,
		Inputs:   tplInputs,
		Local:    true,
	}
	return tpl, nil
}

// KeyIDs produces KeyIDs from a list of xpubs and a derivation path
// (applied to all the xpubs).
func KeyIDs(xpubs []*hd25519.XPub, path []uint32) []KeyID {
	result := make([]KeyID, 0, len(xpubs))
	for _, xpub := range xpubs {
		result = append(result, KeyID{xpub.String(), path})
	}
	return result
}

func Sign(ctx context.Context, tpl *Template, signFn func(context.Context, string, []uint32, [32]byte) ([]byte, error)) error {
	for i, input := range tpl.Inputs {
		for j, c := range input.WitnessComponents {
			err := c.Sign(ctx, tpl, i, signFn)
			if err != nil {
				return errors.WithDetailf(err, "adding signature(s) to witness component %d of input %d", j, i)
			}
		}
	}
	return nil
}
