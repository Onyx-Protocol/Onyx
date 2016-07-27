package issuer

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/errors"
	"chain/metrics"
)

// IssuanceReserver is a txbuilder.Reserver
// that issues an asset
type IssuanceReserver struct {
	bc.AssetID
	AssetDefinition []byte // omit to keep existing asset definition
	ReferenceData   []byte
}

func (ir IssuanceReserver) Reserve(ctx context.Context, amt *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	asset, err := appdb.AssetByID(ctx, ir.AssetID)
	if err != nil {
		return nil, errors.WithDetailf(err, "get asset with ID %q", ir.AssetID)
	}
	in := bc.NewIssuanceInput(time.Now(), time.Now().Add(ttl), asset.GenesisHash, amt.Amount, asset.IssuanceScript, ir.AssetDefinition, ir.ReferenceData, nil)
	return &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{{
			TxInput:       in,
			TemplateInput: issuanceInput(asset, *amt),
		}},
	}, nil
}

// NewIssueSource returns a txbuilder.Source with an IssuanceReserver.
func NewIssueSource(ctx context.Context, assetAmount bc.AssetAmount, assetDefinition, referenceData []byte) *txbuilder.Source {
	return &txbuilder.Source{
		AssetAmount: assetAmount,
		Reserver: IssuanceReserver{
			AssetID:         assetAmount.AssetID,
			AssetDefinition: assetDefinition,
			ReferenceData:   referenceData,
		},
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
		Reserver:    IssuanceReserver{AssetID: assetAmount.AssetID},
	}}

	return txbuilder.Build(ctx, nil, sources, dests, nil, time.Minute)
}

// issuanceInput returns an Input that can be used
// to issue units of asset 'a'.
func issuanceInput(a *appdb.Asset, aa bc.AssetAmount) *txbuilder.Input {
	tmplInp := &txbuilder.Input{AssetAmount: aa}
	sigs := txbuilder.InputSigs(a.Keys, appdb.IssuancePath(a))
	tmplInp.AddWitnessSigs(sigs, txscript.SigsRequired(a.RedeemScript), nil)
	tmplInp.AddWitnessData(a.RedeemScript)
	return tmplInp
}
