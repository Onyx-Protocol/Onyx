// Package txbuilder builds a Chain Protocol transaction from
// a list of actions.
package txbuilder

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sync"
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
	ErrAction              = errors.New("errors occurred in one or more actions")
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

	type res struct {
		buildResult *BuildResult
		err         error
	}
	results := make([]res, len(actions))
	var wg sync.WaitGroup
	wg.Add(len(actions))
	for i := range actions {
		i := i
		go func() {
			defer wg.Done()
			buildResult, err := actions[i].Build(ctx, maxTime)
			results[i] = res{buildResult, err}
		}()
	}

	wg.Wait()
	var (
		tplSigInsts []*SigningInstruction
		errs        []error
		rollbacks   []func()
	)
result:
	for i, v := range results {
		if v.err != nil {
			err := errors.WithData(v.err, "index", i)
			errs = append(errs, err)
			continue result
		}
		buildResult := v.buildResult
		for _, in := range buildResult.Inputs {
			if in.Amount() > math.MaxInt64 {
				err := errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", in.Amount())
				err = errors.WithData(err, "index", i)
				errs = append(errs, err)
				continue result
			}
		}
		for _, out := range buildResult.Outputs {
			if out.Amount > math.MaxInt64 {
				err := errors.WithDetailf(ErrBadAmount, "amount %d exceeds maximum value 2^63", out.Amount)
				err = errors.WithData(err, "index", i)
				errs = append(errs, err)
				continue result
			}
		}

		if len(buildResult.Inputs) != len(buildResult.SigningInstructions) {
			// This would only happen from a bug in our system
			err := errors.Wrap(fmt.Errorf("%T returned different number of inputs and signing instructions", actions[i]))
			err = errors.WithData(err, "index", i)
			errs = append(errs, err)
			continue result
		}

		for i := range buildResult.Inputs {
			buildResult.SigningInstructions[i].Position = len(tx.Inputs)

			// Empty signature arrays should be serialized as empty arrays, not null.
			if buildResult.SigningInstructions[i].WitnessComponents == nil {
				buildResult.SigningInstructions[i].WitnessComponents = []WitnessComponent{}
			}

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

		if buildResult.Rollback != nil {
			rollbacks = append(rollbacks, buildResult.Rollback)
		}
	}

	if len(errs) > 0 {
		rollback(rollbacks)
		return nil, errors.WithData(ErrAction, "actions", errs)
	}

	err := checkBlankCheck(tx)
	if err != nil {
		rollback(rollbacks)
		return nil, err
	}

	if tx.MaxTime == 0 || tx.MaxTime > bc.Millis(maxTime) {
		tx.MaxTime = bc.Millis(maxTime)
	}

	tpl := &Template{
		Transaction:         tx,
		SigningInstructions: tplSigInsts,
		Local:               local,
	}
	return tpl, nil
}

func rollback(rollbacks []func()) {
	for _, f := range rollbacks {
		f()
	}
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
			return errors.WithDetailf(ErrBadAmount, "cumulative amounts for asset %s overflow the allowed asset amount 2^63", asset)
		}
	}
	for _, out := range tx.Outputs {
		assetMap[out.AssetID], ok = checked.SubInt64(assetMap[out.AssetID], int64(out.Amount))
		if !ok {
			return errors.WithDetailf(ErrBadAmount, "cumulative amounts for asset %s overflow the allowed asset amount 2^63", out.AssetID)
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
