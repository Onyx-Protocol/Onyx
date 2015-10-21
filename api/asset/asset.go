// Package asset provides business logic for manipulating assets.
package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/utxodb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain-sandbox/txscript"
	"chain/fedchain/bc"
	"chain/metrics"
)

// ErrBadAddr is returned by Issue.
var ErrBadAddr = errors.New("bad address")

// Issue creates a transaction that
// issues new units of an asset
// distributed to the outputs provided.
func Issue(ctx context.Context, assetID string, outs []*Output) (*Tx, error) {
	defer metrics.RecordElapsed(time.Now())
	tx := &bc.Tx{Version: bc.CurrentTransactionVersion}
	tx.Inputs = append(tx.Inputs, &bc.TxInput{Previous: bc.IssuanceOutpoint})

	hash, err := bc.ParseHash(assetID)
	assetHash := bc.AssetID(hash)

	asset, err := appdb.AssetByID(ctx, assetHash)
	if err != nil {
		return nil, errors.WithDetailf(err, "get asset with ID %q", assetID)
	}

	for i, out := range outs {
		if (out.AccountID == "") == (out.Address == "") {
			return nil, errors.WithDetailf(ErrBadOutDest, "output index=%d", i)
		}
	}

	outRecvs, err := addAssetIssuanceOutputs(ctx, tx, asset, outs)
	if err != nil {
		return nil, errors.Wrap(err, "add issuance outputs")
	}

	appTx := &Tx{
		Unsigned:   tx,
		BlockChain: "sandbox", // TODO(tess): make this BlockChain: blockchain.FromContext(ctx)
		Inputs:     []*Input{issuanceInput(asset, tx)},
		OutRecvs:   outRecvs,
	}
	return appTx, nil
}

// Output is a user input struct that describes
// the destination of a transaction's inputs.
type Output struct {
	AssetID   string `json:"asset_id"`
	Address   string `json:"address"`
	AccountID string `json:"account_id"`
	Amount    uint64 `json:"amount"`
	isChange  bool
}

// PKScript returns the script for sending to
// the destination address or account id provided.
// For an Address-type output, the returned *utxodb.Receiver is nil.
func (o *Output) PKScript(ctx context.Context) ([]byte, *utxodb.Receiver, error) {
	if o.AccountID != "" {
		addr := &appdb.Address{
			AccountID: o.AccountID,
			IsChange:  o.isChange,
		}
		err := CreateAddress(ctx, addr, false)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "output create address error account=%v", o.AccountID)
		}
		return addr.PKScript, newOutputReceiver(addr, o.isChange), nil
	}
	script, err := txscript.AddrPkScript(o.Address)
	if err != nil {
		return nil, nil, errors.Wrapf(ErrBadAddr, "output pkscript error addr=%v", o.Address)
	}
	return script, nil, nil
}

func addAssetIssuanceOutputs(ctx context.Context, tx *bc.Tx, asset *appdb.Asset, outs []*Output) ([]*utxodb.Receiver, error) {
	var outAddrs []*utxodb.Receiver
	for i, out := range outs {
		pkScript, receiver, err := out.PKScript(ctx)
		if err != nil {
			return nil, errors.WithDetailf(err, "output %d", i)
		}

		tx.Outputs = append(tx.Outputs, &bc.TxOutput{AssetID: asset.Hash, Value: out.Amount, Script: pkScript})
		outAddrs = append(outAddrs, receiver)
	}
	return outAddrs, nil
}

func newOutputReceiver(addr *appdb.Address, isChange bool) *utxodb.Receiver {
	return &utxodb.Receiver{
		ManagerNodeID: addr.ManagerNodeID,
		AccountID:     addr.AccountID,
		AddrIndex:     addr.Index,
		IsChange:      isChange,
	}
}

// issuanceInput returns an Input that can be used
// to issue units of asset 'a'.
func issuanceInput(a *appdb.Asset, tx *bc.Tx) *Input {
	return &Input{
		IssuerNodeID:  a.IssuerNodeID,
		RedeemScript:  a.RedeemScript,
		SignatureData: tx.Hash(),
		Sigs:          inputSigs(hdkey.Derive(a.Keys, appdb.IssuancePath(a))),
	}
}

func inputSigs(keys []*hdkey.Key) (sigs []*Signature) {
	for _, k := range keys {
		sigs = append(sigs, &Signature{
			XPub:           k.Root.String(),
			DerivationPath: k.Path,
		})
	}
	return sigs
}
