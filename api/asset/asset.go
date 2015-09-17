// Package asset provides business logic for manipulating assets.
package asset

import (
	"bytes"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain-sandbox/txscript"
	"chain/fedchain-sandbox/wire"
	"chain/metrics"
)

// ErrBadAddr is returned by Issue.
var ErrBadAddr = errors.New("bad address")

// Issue creates a transaction that
// issues new units of an asset
// distributed to the outputs provided.
func Issue(ctx context.Context, assetID string, outs []Output) (*Tx, error) {
	defer metrics.RecordElapsed(time.Now())
	tx := wire.NewMsgTx()
	tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}))

	asset, err := appdb.AssetByID(ctx, assetID)
	if err != nil {
		return nil, errors.WithDetailf(err, "get asset with ID %q", assetID)
	}

	for i, out := range outs {
		if (out.BucketID == "") == (out.Address == "") {
			return nil, errors.WithDetailf(ErrBadOutDest, "output index=%d", i)
		}
	}

	err = addAssetIssuanceOutputs(ctx, tx, asset, outs)
	if err != nil {
		return nil, errors.Wrap(err, "add issuance outputs")
	}

	var buf bytes.Buffer
	tx.Serialize(&buf)
	appTx := &Tx{
		Unsigned:   buf.Bytes(),
		BlockChain: "sandbox", // TODO(tess): make this BlockChain: blockchain.FromContext(ctx)
		Inputs:     []*Input{issuanceInput(asset, tx)},
	}
	return appTx, nil
}

// Output is a user input struct that describes
// the destination of a transaction's inputs.
type Output struct {
	AssetID  string `json:"asset_id"`
	Address  string `json:"address"`
	BucketID string `json:"bucket_id"`
	Amount   int64  `json:"amount"`
	isChange bool
}

// PkScript returns the script for sending to
// the destination address or bucket id provided.
func (o *Output) PkScript(ctx context.Context) ([]byte, error) {
	if o.BucketID != "" {
		addr := &appdb.Address{
			BucketID: o.BucketID,
		}
		err := CreateAddress(ctx, addr)
		if err != nil {
			return nil, errors.Wrapf(err, "output create address error bucket=%v", o.BucketID)
		}
		return addr.PKScript, nil
	}
	script, err := txscript.AddrPkScript(o.Address)
	if err != nil {
		return nil, errors.Wrapf(ErrBadAddr, "output pkscript error addr=%v", o.Address)
	}
	return script, nil
}

func addAssetIssuanceOutputs(ctx context.Context, tx *wire.MsgTx, asset *appdb.Asset, outs []Output) error {
	for i, out := range outs {
		pkScript, err := out.PkScript(ctx)
		if err != nil {
			return errors.WithDetailf(err, "output %d: %v", i, err.Error())
		}

		tx.AddTxOut(wire.NewTxOut(asset.Hash, out.Amount, pkScript))
	}
	return nil
}

// issuanceInput returns an Input that can be used
// to issue units of asset 'a'.
func issuanceInput(a *appdb.Asset, tx *wire.MsgTx) *Input {
	var buf bytes.Buffer
	tx.Serialize(&buf)

	return &Input{
		AssetGroupID:  a.GroupID,
		RedeemScript:  a.RedeemScript,
		SignatureData: wire.DoubleSha256(buf.Bytes()),
		Sigs:          inputSigs(Signers(a.Keys, IssuancePath(a))),
	}
}

func inputSigs(keys []*DerivedKey) (sigs []*Signature) {
	for _, k := range keys {
		sigs = append(sigs, &Signature{
			XPub:           k.Root.String(),
			DerivationPath: k.Path,
		})
	}
	return sigs
}
