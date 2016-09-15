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
)

// Build builds or adds on to a transaction.
// Initially, inputs are left unconsumed, and destinations unsatisfied.
// Build partners then satisfy and consume inputs and destinations.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Build(ctx context.Context, tx *bc.TxData, actions []Action, ref json.Map, maxTime time.Time) (*Template, error) {
	var local bool
	if tx == nil {
		tx = &bc.TxData{
			Version: bc.CurrentTransactionVersion,
		}
		local = true
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

	var tplSigInsts []*SigningInstruction
	for i, action := range actions {
		txins, txouts, sigInsts, err := action.Build(ctx, maxTime)
		if err != nil {
			return nil, errors.WithDetailf(err, "invalid action %d", i)
		}

		if len(txins) != len(sigInsts) {
			// This would only happen from a bug in our system
			return nil, errors.Wrap(fmt.Errorf("%T returned different number of inputs and signing instructions", action))
		}

		for i := range txins {
			sigInsts[i].Position = uint32(len(tx.Inputs))
			tplSigInsts = append(tplSigInsts, sigInsts[i])
			tx.Inputs = append(tx.Inputs, txins[i])
		}

		tx.Outputs = append(tx.Outputs, txouts...)
	}

	for _, sigInst := range tplSigInsts {
		// Empty signature arrays should be serialized as empty arrays, not null.
		if sigInst.WitnessComponents == nil {
			sigInst.WitnessComponents = []WitnessComponent{}
		}
	}

	tpl := &Template{
		Transaction:         tx,
		SigningInstructions: tplSigInsts,
		Local:               local,
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
	for i, sigInst := range tpl.SigningInstructions {
		for j, c := range sigInst.WitnessComponents {
			err := c.Sign(ctx, tpl, i, signFn)
			if err != nil {
				return errors.WithDetailf(err, "adding signature(s) to witness component %d of input %d", j, i)
			}
		}
	}
	return materializeWitnesses(tpl)
}
