package asset

import (
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	chaintxscript "chain/fedchain-sandbox/txscript"
	"chain/metrics"
)

// Create generates a new asset redeem script
// and id inside of an asset group.
func Create(ctx context.Context, agID, label string) (*appdb.Asset, error) {
	defer metrics.RecordElapsed(time.Now())
	if label == "" {
		return nil, appdb.ErrBadLabel
	}

	asset, sigsReq, err := appdb.NextAsset(ctx, agID)
	if err != nil {
		return nil, errors.Wrap(err, "getting asset key info")
	}

	asset.Label = label

	var pubkeys []*btcutil.AddressPubKey
	for _, key := range hdkey.Derive(asset.Keys, appdb.IssuancePath(asset)) {
		pubkeys = append(pubkeys, key.Address)
	}

	asset.RedeemScript, err = txscript.MultiSigScript(pubkeys, sigsReq)
	if err != nil {
		return nil, errors.Wrapf(err, "creating asset: group id %v sigsReq %v", agID, sigsReq)
	}
	pkScript, err := chaintxscript.RedeemToPkScript(asset.RedeemScript)
	if err != nil {
		return nil, err
	}
	asset.Hash = chaintxscript.PkScriptToAssetID(pkScript)

	err = appdb.InsertAsset(ctx, asset)
	if err != nil {
		return nil, errors.Wrap(err, "inserting asset")
	}

	return asset, nil
}
