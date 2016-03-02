package issuer

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/hdkey"
	"chain/fedchain/txscript"
	"chain/metrics"
)

// IssuanceReserver is a txbuilder.Reserver
// that issues an asset
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

		newDefHash := bc.HashAssetDefinition(ir.asset.Definition).String()
		if defHash != newDefHash {
			in.AssetDefinition = ir.asset.Definition
		}
	}
	return &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{{
			TxInput:       in,
			TemplateInput: issuanceInput(ir.asset, *amt),
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
func issuanceInput(a *appdb.Asset, aa bc.AssetAmount) *txbuilder.Input {
	return &txbuilder.Input{
		AssetAmount:     aa,
		SigScriptSuffix: txscript.AddDataToScript(nil, a.RedeemScript),
		Sigs:            txbuilder.InputSigs(hdkey.Derive(a.Keys, appdb.IssuancePath(a))),
	}
}
