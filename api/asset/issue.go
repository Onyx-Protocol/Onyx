// Package asset provides business logic for manipulating assets.
package asset

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/crypto/hash256"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
	"chain/metrics"
)

type IssuanceReserver struct {
	asset *appdb.Asset
}

func (ir *IssuanceReserver) Reserve(ctx context.Context, amt *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	in := &bc.TxInput{
		Previous: bc.Outpoint{
			Index: bc.InvalidOutputIndex,
			Hash:  bc.Hash{}, // TODO(kr): figure out anti-replay for issuance
		},
	}
	if len(ir.asset.Definition) != 0 {
		defHash, err := txdb.DefinitionHashByAssetID(ctx, ir.asset.Hash.String())
		if err != nil && errors.Root(err) != sql.ErrNoRows {
			return nil, errors.WithDetailf(err, "get asset definition pointer for %s", ir.asset.Hash)
		}

		newDefHash := bc.Hash(hash256.Sum(ir.asset.Definition)).String()
		if defHash != newDefHash {
			in.AssetDefinition = ir.asset.Definition
		}
	}
	return &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{{
			TxInput:       in,
			TemplateInput: issuanceInput(ir.asset),
		}},
	}, nil
}

// Issue creates a transaction that
// issues new units of an asset
// distributed to the outputs provided.
func Issue(ctx context.Context, assetID string, dests []*txbuilder.Destination) (*txbuilder.Template, error) {
	defer metrics.RecordElapsed(time.Now())

	hash, err := bc.ParseHash(assetID)
	assetHash := bc.AssetID(hash)

	asset, err := appdb.AssetByID(ctx, assetHash)
	if err != nil {
		return nil, errors.WithDetailf(err, "get asset with ID %q", assetID)
	}

	sources := []*txbuilder.Source{{
		Reserver: &IssuanceReserver{asset: asset},
	}}

	return txbuilder.Build(ctx, nil, sources, dests, nil, time.Minute)
}

// issuanceInput returns an Input that can be used
// to issue units of asset 'a'.
func issuanceInput(a *appdb.Asset) *txbuilder.Input {
	return &txbuilder.Input{
		SigScriptSuffix: txscript.AddDataToScript(nil, a.RedeemScript),
		Sigs:            inputSigs(hdkey.Derive(a.Keys, appdb.IssuancePath(a))),
	}
}

func inputSigs(keys []*hdkey.Key) (sigs []*txbuilder.Signature) {
	for _, k := range keys {
		sigs = append(sigs, &txbuilder.Signature{
			XPub:           k.Root.String(),
			DerivationPath: k.Path,
		})
	}
	return sigs
}
