package asset

import (
	"context"
	"time"

	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vmutil"
)

type IssueAction struct {
	Params struct {
		bc.AssetAmount
		TTL     time.Duration
		MinTime *time.Time `json:"min_time"`

		// This field is only necessary for filtering
		// aliases on transaction build requests. A wrapper
		// function reads it to set the ID field. It is
		// not used anywhere else in the code base.
		AssetAlias string `json:"asset_alias"`
	}
	ReferenceData json.Map `json:"reference_data"`
}

func (a *IssueAction) Build(ctx context.Context) ([]*bc.TxInput, []*bc.TxOutput, []*txbuilder.Input, error) {
	asset, err := FindByID(ctx, a.Params.AssetID)
	if errors.Root(err) == pg.ErrUserInputNotFound {
		err = errors.WithDetailf(err, "missing asset with ID %q", a.Params.AssetID)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	ttl := a.Params.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	minTime := time.Now()
	if a.Params.MinTime != nil {
		minTime = *a.Params.MinTime
	}
	txin := bc.NewIssuanceInput(minTime, minTime.Add(ttl), asset.InitialBlockHash, a.Params.Amount, asset.IssuanceProgram, a.ReferenceData, nil)
	tplIn := issuanceInput(asset, a.Params.AssetAmount)

	return []*bc.TxInput{txin}, nil, []*txbuilder.Input{tplIn}, nil
}

// issuanceInput returns an Input that can be used
// to issue units of asset 'a'.
func issuanceInput(a *Asset, aa bc.AssetAmount) *txbuilder.Input {
	tmplInp := &txbuilder.Input{AssetAmount: aa}
	path := signers.Path(a.Signer, signers.AssetKeySpace, nil)
	sigs := txbuilder.InputSigs(a.Signer.XPubs, path)
	tmplInp.AddWitnessSigs(sigs, vmutil.SigsRequired(a.IssuanceProgram), nil)
	return tmplInp
}
