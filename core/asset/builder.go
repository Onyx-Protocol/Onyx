package asset

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"time"

	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
)

func (reg *Registry) NewIssueAction(assetAmount bc.AssetAmount, referenceData chainjson.Map) txbuilder.Action {
	return &issueAction{
		assets:        reg,
		TTL:           chainjson.Duration{24 * time.Hour},
		AssetAmount:   assetAmount,
		ReferenceData: referenceData,
	}
}

func (reg *Registry) DecodeIssueAction(data []byte) (txbuilder.Action, error) {
	a := &issueAction{assets: reg}
	err := json.Unmarshal(data, a)
	return a, err
}

type issueAction struct {
	assets *Registry
	bc.AssetAmount
	TTL           chainjson.Duration
	ReferenceData chainjson.Map `json:"reference_data"`
}

func (a *issueAction) Build(ctx context.Context) (*txbuilder.BuildResult, error) {
	now := time.Now()

	// Auto-supply a nonzero mintime that allows for some clock skew
	// between this computer and whatever machine validates the
	// transaction.
	minTime := now.Add(-5 * time.Minute)

	ttl := a.TTL.Duration
	if ttl == 0 {
		ttl = time.Minute
	}
	maxTime := now.Add(ttl)

	asset, err := a.assets.findByID(ctx, a.AssetID)
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
