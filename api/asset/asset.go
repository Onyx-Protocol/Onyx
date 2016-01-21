// Package asset provides business logic for manipulating assets.
package asset

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/crypto/hash256"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
	"chain/metrics"
)

// Issue creates a transaction that
// issues new units of an asset
// distributed to the outputs provided.
func Issue(ctx context.Context, assetID string, dests []*Destination) (*TxTemplate, error) {
	defer metrics.RecordElapsed(time.Now())

	hash, err := bc.ParseHash(assetID)
	assetHash := bc.AssetID(hash)

	asset, err := appdb.AssetByID(ctx, assetHash)
	if err != nil {
		return nil, errors.WithDetailf(err, "get asset with ID %q", assetID)
	}

	tx := &bc.TxData{Version: bc.CurrentTransactionVersion}
	in := &bc.TxInput{Previous: bc.Outpoint{
		Index: bc.InvalidOutputIndex,
		Hash:  bc.Hash{}, // TODO(kr): figure out anti-replay for issuance
	}}

	if len(asset.Definition) != 0 {
		defHash, err := txdb.DefinitionHashByAssetID(ctx, assetID)
		if err != nil && errors.Root(err) != sql.ErrNoRows {
			return nil, errors.WithDetailf(err, "get asset definition pointer for %s", assetID)
		}

		newDefHash := bc.Hash(hash256.Sum(asset.Definition)).String()
		if defHash != newDefHash {
			in.AssetDefinition = asset.Definition
		}
	}

	tx.Inputs = append(tx.Inputs, in)

	addAssetIssuanceOutputs(ctx, tx, asset, dests)

	input, err := issuanceInput(asset, tx)
	if err != nil {
		return nil, errors.Wrap(err, "creating issuance Input")
	}

	receivers := make([]Receiver, 0, len(dests))
	for _, dest := range dests {
		receivers = append(receivers, dest.Receiver)
	}

	appTx := &TxTemplate{
		Unsigned:   tx,
		BlockChain: "sandbox", // TODO(tess): make this BlockChain: blockchain.FromContext(ctx)
		Inputs:     []*Input{input},
		OutRecvs:   receivers,
	}
	return appTx, nil
}

func addAssetIssuanceOutputs(ctx context.Context, tx *bc.TxData, asset *appdb.Asset, dests []*Destination) {
	for _, dest := range dests {
		tx.Outputs = append(tx.Outputs, &bc.TxOutput{
			AssetID:  asset.Hash,
			Value:    dest.Amount,
			Script:   dest.PKScript(),
			Metadata: dest.Metadata,
		})
	}
}

// issuanceInput returns an Input that can be used
// to issue units of asset 'a'.
func issuanceInput(a *appdb.Asset, tx *bc.TxData) (*Input, error) {
	hash, err := txscript.CalcSignatureHash(tx, 0, a.RedeemScript, txscript.SigHashAll)
	if err != nil {
		return nil, errors.Wrap(err, "calculating signature hash")
	}

	return &Input{
		RedeemScript:  a.RedeemScript,
		SignatureData: hash,
		Sigs:          inputSigs(hdkey.Derive(a.Keys, appdb.IssuancePath(a))),
	}, nil
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
