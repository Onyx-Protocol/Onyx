package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/errors"
)

// IssuanceReserver is a txbuilder.Reserver
// that issues an asset
type IssuanceReserver struct {
	bc.AssetID
	ReferenceData []byte
}

func (ir IssuanceReserver) Reserve(ctx context.Context, amt *bc.AssetAmount, ttl time.Duration) (*txbuilder.ReserveResult, error) {
	asset, err := Find(ctx, ir.AssetID)
	if err != nil {
		return nil, errors.WithDetailf(err, "find asset with ID %q", ir.AssetID)
	}

	in := bc.NewIssuanceInput(time.Now(), time.Now().Add(ttl), asset.GenesisHash, amt.Amount, asset.IssuanceProgram, ir.ReferenceData, nil)
	return &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{{
			TxInput:       in,
			TemplateInput: issuanceInput(asset, *amt),
		}},
	}, nil
}

// NewIssueSource returns a txbuilder.Source with an IssuanceReserver.
func NewIssueSource(ctx context.Context, assetAmount bc.AssetAmount, referenceData []byte) *txbuilder.Source {
	return &txbuilder.Source{
		AssetAmount: assetAmount,
		Reserver: IssuanceReserver{
			AssetID:       assetAmount.AssetID,
			ReferenceData: referenceData,
		},
	}
}

// issuanceInput returns an Input that can be used
// to issue units of asset 'a'.
func issuanceInput(a *Asset, aa bc.AssetAmount) *txbuilder.Input {
	tmplInp := &txbuilder.Input{AssetAmount: aa}
	path := signers.Path(a.Signer, signers.AssetKeySpace, a.KeyIndex) // is this the right key index?
	sigs := txbuilder.InputSigs(a.Signer.XPubs, path)
	tmplInp.AddWitnessSigs(sigs, txscript.SigsRequired(a.IssuanceProgram), nil)
	return tmplInp
}
