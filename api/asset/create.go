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
	"chain/fedchain/bc"
	"chain/fedchain/validation"
	"chain/metrics"
)

// Create generates a new asset redeem script
// and id inside of an issuer node.
func Create(ctx context.Context, inodeID, label string) (*appdb.Asset, error) {
	defer metrics.RecordElapsed(time.Now())
	if label == "" {
		return nil, appdb.ErrBadLabel
	}

	asset, sigsReq, err := appdb.NextAsset(ctx, inodeID)
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
		return nil, errors.Wrapf(err, "creating asset: issuer node id %v sigsReq %v", inodeID, sigsReq)
	}
	pkScript := chaintxscript.RedeemToPkScript(asset.RedeemScript)
	asset.Hash = bc.ComputeAssetID(pkScript, validation.TestParams.GenesisHash)

	err = appdb.InsertAsset(ctx, asset)
	if err != nil {
		return nil, errors.Wrap(err, "inserting asset")
	}

	return asset, nil
}
