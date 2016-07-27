package issuer

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/crypto/ed25519/hd25519"
	"chain/errors"
	"chain/metrics"
)

// CreateAsset generates a new asset redeem script
// and id inside of an issuer node.
// TODO(jackson): Once SDKs have been adopted and everyone has updated,
// we should make clientToken required.
func CreateAsset(ctx context.Context, inodeID, label string, genesisHash bc.Hash, definition map[string]interface{}, clientToken *string) (*appdb.Asset, error) {
	defer metrics.RecordElapsed(time.Now())
	if label == "" {
		return nil, errors.WithDetail(appdb.ErrBadLabel, "missing/null value")
	}

	asset, sigsReq, err := appdb.NextAsset(ctx, inodeID)
	if err != nil {
		return nil, errors.Wrap(err, "getting asset key info")
	}

	asset.ClientToken = clientToken
	asset.Label = label
	asset.Definition, err = serializeAssetDef(definition)
	if err != nil {
		return nil, errors.Wrap(err, "serializing asset definition")
	}

	derivedXPubs := hd25519.DeriveXPubs(asset.Keys, appdb.IssuancePath(asset))
	derivedPubs := hd25519.XPubKeys(derivedXPubs)
	pkScript, redeem, err := txscript.Scripts(derivedPubs, sigsReq)
	if err != nil {
		return nil, errors.Wrapf(err, "creating asset: asset issuer id %v sigsReq %v", inodeID, sigsReq)
	}
	asset.IssuanceScript = pkScript
	asset.RedeemScript = redeem
	asset.GenesisHash = genesisHash
	asset.Hash = bc.ComputeAssetID(pkScript, genesisHash, 1)

	asset, err = appdb.InsertAsset(ctx, asset)
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
	if def == nil {
		return nil, nil
	}
	return json.MarshalIndent(def, "", "  ")
}
