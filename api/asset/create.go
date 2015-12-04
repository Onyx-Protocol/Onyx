package asset

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"

	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	chaintxscript "chain/fedchain-sandbox/txscript"
	"chain/fedchain/bc"
	"chain/metrics"
)

// Create generates a new asset redeem script
// and id inside of an issuer node.
func Create(ctx context.Context, inodeID, label string, definition map[string]interface{}) (*appdb.Asset, error) {
	defer metrics.RecordElapsed(time.Now())
	if label == "" {
		return nil, appdb.ErrBadLabel
	}

	if definition == nil {
		// Definitions can be empty (`{}`) but they cannot be nil.
		return nil, ErrBadDefinition
	}

	asset, sigsReq, err := appdb.NextAsset(ctx, inodeID)
	if err != nil {
		return nil, errors.Wrap(err, "getting asset key info")
	}

	asset.Label = label
	asset.Definition, err = serializeAssetDef(definition)
	if err != nil {
		return nil, errors.Wrap(err, "serializing asset definition")
	}

	var pubkeys []*btcutil.AddressPubKey
	for _, key := range hdkey.Derive(asset.Keys, appdb.IssuancePath(asset)) {
		pubkeys = append(pubkeys, key.Address)
	}

	asset.RedeemScript, err = txscript.MultiSigScript(pubkeys, sigsReq)
	if err != nil {
		return nil, errors.Wrapf(err, "creating asset: issuer node id %v sigsReq %v", inodeID, sigsReq)
	}
	pkScript := chaintxscript.RedeemToPkScript(asset.RedeemScript)
	asset.Hash = bc.ComputeAssetID(pkScript, [32]byte{}) // TODO(kr): get genesis hash from config
	asset.IssuanceScript = pkScript

	err = appdb.InsertAsset(ctx, asset)
	if err != nil {
		return nil, errors.Wrap(err, "inserting asset")
	}

	return asset, nil
}

// serializeAssetDef produces a canonical byte representation of an asset
// definition. Currently, this is implemented using pretty-printed JSON.
// As is the standard for Go's map[string] serialization, object keys will
// appear in lexicographic order. Although this is mostly meant for machine
// consumption, the JSON is pretty-printed for easy reading.
func serializeAssetDef(def map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(def, "", "  ")
}
