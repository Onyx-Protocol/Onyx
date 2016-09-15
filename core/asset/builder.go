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
	bc.AssetAmount
	TTL     time.Duration
	MinTime *time.Time `json:"min_time"`

	// This field is only necessary for filtering
	// aliases on transaction build requests. A wrapper
	// function reads it to set the ID field. It is
	// not used anywhere else in the code base.
	AssetAlias string `json:"asset_alias"`

	ReferenceData json.Map `json:"reference_data"`
}

func (a IssueAction) GetTTL() time.Duration {
	ttl := a.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	return ttl
}

func (a *IssueAction) Build(ctx context.Context, maxTime time.Time) (
	[]*bc.TxInput,
	[]*bc.TxOutput,
	[]*txbuilder.SigningInstruction,
	error,
) {
	asset, err := FindByID(ctx, a.AssetID)
	if errors.Root(err) == pg.ErrUserInputNotFound {
		err = errors.WithDetailf(err, "missing asset with ID %q", a.AssetID)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	minTime := time.Now()
	if a.MinTime != nil {
		minTime = *a.MinTime
	}
	txin := bc.NewIssuanceInput(minTime, maxTime, asset.InitialBlockHash, a.Amount, asset.IssuanceProgram, a.ReferenceData, nil)

	tplIn := &txbuilder.SigningInstruction{AssetAmount: a.AssetAmount}
	path := signers.Path(asset.Signer, signers.AssetKeySpace, nil)
	keyIDs := txbuilder.KeyIDs(asset.Signer.XPubs, path)
	_, nrequired, err := vmutil.ParseP2DPMultiSigProgram(asset.IssuanceProgram)
	if err != nil {
		return nil, nil, nil, err
	}

	tplIn.AddWitnessKeys(keyIDs, nrequired)

	return []*bc.TxInput{txin}, nil, []*txbuilder.SigningInstruction{tplIn}, nil
}
