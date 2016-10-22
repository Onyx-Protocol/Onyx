// Package txbuilder builds a Chain Protocol transaction from
// a list of actions.
package txbuilder

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"time"

	"chain/crypto/ed25519/chainkd"
	"chain/encoding/json"
	"chain/errors"
	"chain/math/checked"
	"chain/protocol/bc"
)

var (
	ErrBadRefData          = errors.New("transaction reference data does not match previous template's reference data")
	ErrBadTxInputIdx       = errors.New("unsigned tx missing input")
	ErrBadWitnessComponent = errors.New("invalid witness component")
	ErrBadAmount           = errors.New("bad asset amount")
	ErrBlankCheck          = errors.New("unsafe transaction: leaves assets free to control")
)

// Build builds or adds on to a transaction.
// Initially, inputs are left unconsumed, and destinations unsatisfied.
// Build partners then satisfy and consume inputs and destinations.
// The final party must ensure that the transaction is
// balanced before calling finalize.
func Build(ctx context.Context, tx *bc.TxData, actions []Action, maxTime time.Time) (*Template, error) {
	var local bool
	if tx == nil {
		tx = &bc.TxData{
			Version: bc.CurrentTransactionVersion,
		}
		local = true
	}

	var tplSigInsts []*SigningInstruction
	for i, action := range actions {
		buildResult, err := action.Build(ctx, maxTime)
		if err != nil {
			return nil, errors.WithDetailf(err, "invalid action %d", i)
		}

		for _, in := range buildResult.Inputs {
			if in.Amount() > math.MaxInt64 {
				return nil, errors.WithDetailf(ErrBadAmount, "bad amount %d for action %d", in.Amount(), i)
			}
		}
		for _, out := range buildResult.Outputs {
			if out.Amount > math.MaxInt64 {
				return nil, errors.WithDetailf(ErrBadAmount, "bad amount %d for action %d", out.Amount, i)
			}
		}

		if len(buildResult.Inputs) != len(buildResult.SigningInstructions) {
			// This would only happen from a bug in our system
			return nil, errors.Wrap(fmt.Errorf("%T returned different number of inputs and signing instructions", action))
		}

		for i := range buildResult.Inputs {
			buildResult.SigningInstructions[i].Position = len(tx.Inputs)
			tplSigInsts = append(tplSigInsts, buildResult.SigningInstructions[i])
			tx.Inputs = append(tx.Inputs, buildResult.Inputs[i])
		}

		tx.Outputs = append(tx.Outputs, buildResult.Outputs...)

		if len(buildResult.ReferenceData) > 0 {
			if len(tx.ReferenceData) != 0 && !bytes.Equal(tx.ReferenceData, buildResult.ReferenceData) {
				// There can be only one! ...caller that sets reference data
				return nil, errors.Wrap(ErrBadRefData)
			}
			tx.ReferenceData = buildResult.ReferenceData
		}

		if buildResult.MinTimeMS > 0 {
			if buildResult.MinTimeMS > tx.MinTime {
				tx.MinTime = buildResult.MinTimeMS
			}
		}
	}

	err := checkBlankCheck(tx)
	if err != nil {
		return nil, err
	}

	if tx.MaxTime == 0 || tx.MaxTime > bc.Millis(maxTime) {
		tx.MaxTime = bc.Millis(maxTime)
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
func KeyIDs(xpubs []chainkd.XPub, path [][]byte) []KeyID {
	result := make([]KeyID, 0, len(xpubs))
	var hexPath []json.HexBytes
	for _, p := range path {
		hexPath = append(hexPath, p)
	}
	for _, xpub := range xpubs {
		result = append(result, KeyID{xpub.String(), hexPath})
	}
	return result
}

func Sign(ctx context.Context, tpl *Template, xpubs []string, signFn SignFunc) error {
	for i, sigInst := range tpl.SigningInstructions {
		for j, c := range sigInst.WitnessComponents {
			err := c.Sign(ctx, tpl, i, xpubs, signFn)
			if err != nil {
				return errors.WithDetailf(err, "adding signature(s) to witness component %d of input %d", j, i)
			}
		}
	}
	return materializeWitnesses(tpl)
}

func checkBlankCheck(tx *bc.TxData) error {
	assetMap := make(map[bc.AssetID]int64)
	var ok bool
	for _, in := range tx.Inputs {
		asset := in.AssetID() // AssetID() is calculated for IssuanceInputs, so grab once
		assetMap[asset], ok = checked.AddInt64(assetMap[asset], int64(in.Amount()))
		if !ok {
			return errors.WithDetailf(ErrBadAmount, "amounts for asset %s overflow the allowed asset amount", asset)
		}
	}
	for _, out := range tx.Outputs {
		assetMap[out.AssetID], ok = checked.SubInt64(assetMap[out.AssetID], int64(out.Amount))
		if !ok {
			return errors.WithDetailf(ErrBadAmount, "amounts for asset %s overflow the allowed asset amount", out.AssetID)
		}
	}

	var requiresOutputs, requiresInputs bool
	for _, amt := range assetMap {
		if amt > 0 {
			requiresOutputs = true
		}
		if amt < 0 {
			requiresInputs = true
		}
	}

	// 4 possible cases here:
	// 1. requiresOutputs - false requiresInputs - false
	//    This is a balanced transaction with no free assets to consume.
	//    It could potentially be a complete transaction.
	// 2. requiresOutputs - true requiresInputs - false
	//    This is an unbalanced transaction with free assets to consume
	// 3. requiresOutputs - false requiresInputs - true
	//    This is an unbalanced transaction with a requiring assets to be spent
	// 4. requiresOutputs - true requiresInputs - true
	//    This is an unbalanced transaction with free assets to consume
	//    and requiring assets to be spent.
	// The only case that needs to be protected against is 2.
	if requiresOutputs && !requiresInputs {
		return errors.Wrap(ErrBlankCheck)
	}

	return nil
}
