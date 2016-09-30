package asset

import (
	"context"
	"crypto/rand"
	"time"

	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
)

type IssueAction struct {
	bc.AssetAmount
	TTL time.Duration

	// This field is only necessary for filtering
	// aliases on transaction build requests. A wrapper
	// function reads it to set the ID field. It is
	// not used anywhere else in the code base.
	AssetAlias string `json:"asset_alias"`

	ReferenceData json.Map `json:"reference_data"`
}

func (a *IssueAction) Build(ctx context.Context) (*txbuilder.BuildResult, error) {
	now := time.Now()

	// Auto-supply a nonzero mintime that allows for some clock skew
	// between this computer and whatever machine validates the
	// transaction.
	minTime := now.Add(-5 * time.Minute)

	ttl := a.TTL
	if ttl == 0 {
		ttl = time.Minute
	}
	maxTime := now.Add(ttl)

	asset, err := FindByID(ctx, a.AssetID)
	if errors.Root(err) == pg.ErrUserInputNotFound {
		err = errors.WithDetailf(err, "missing asset with ID %q", a.AssetID)
	}
	if err != nil {
		return nil, err
	}

	var nonce [8]byte
	_, err = rand.Read(nonce[:])
	if err != nil {
		return nil, err
	}
	txin := bc.NewIssuanceInput(nonce[:], a.Amount, a.ReferenceData, asset.InitialBlockHash, asset.IssuanceProgram, nil)

	tplIn := &txbuilder.SigningInstruction{AssetAmount: a.AssetAmount}
	path := signers.Path(asset.Signer, signers.AssetKeySpace)
	keyIDs := txbuilder.KeyIDs(asset.Signer.XPubs, path)
	tplIn.AddWitnessKeys(keyIDs, asset.Signer.Quorum)

	return &txbuilder.BuildResult{
		Inputs:              []*bc.TxInput{txin},
		SigningInstructions: []*txbuilder.SigningInstruction{tplIn},
		MinTimeMS:           bc.Millis(minTime),
		MaxTimeMS:           bc.Millis(maxTime),
	}, nil
}
