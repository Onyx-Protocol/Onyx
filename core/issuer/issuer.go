package issuer

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/cos/txscript"
	"chain/errors"
	"chain/metrics"
)

// IssuanceReserver is a txbuilder.Reserver
// that issues an asset
type IssuanceReserver bc.AssetID

func (ir IssuanceReserver) Reserve(ctx context.Context, amt *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	asset, err := appdb.AssetByID(ctx, bc.AssetID(ir))
	if err != nil {
		return nil, errors.WithDetailf(err, "get asset with ID %q", bc.AssetID(ir))
	}

	in := &bc.TxInput{
		Previous: bc.Outpoint{
			Index: bc.InvalidOutputIndex,
			Hash:  bc.Hash{}, // TODO(kr): figure out anti-replay for issuance
		},
		AssetAmount: *amt,
	}
	if len(asset.Definition) != 0 {
		in.AssetDefinition = asset.Definition
	}
	return &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{{
			TxInput:       in,
			TemplateInput: issuanceInput(asset, *amt),
		}},
	}, nil
}

// NewIssueSource returns a txbuilder.Source with an IssuanceReserver.
func NewIssueSource(ctx context.Context, assetAmount *bc.AssetAmount) *txbuilder.Source {
	return &txbuilder.Source{
		AssetAmount: *assetAmount,
		Reserver:    IssuanceReserver(assetAmount.AssetID),
	}
}

// Issue creates a transaction that
// issues new units of an asset
// distributed to the outputs provided.
// DEPRECATED
func Issue(ctx context.Context, assetAmount bc.AssetAmount, dests []*txbuilder.Destination) (*txbuilder.Template, error) {
	defer metrics.RecordElapsed(time.Now())

	sources := []*txbuilder.Source{{
		AssetAmount: assetAmount,
		Reserver:    IssuanceReserver(assetAmount.AssetID),
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
