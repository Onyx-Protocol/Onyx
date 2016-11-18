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
	ReferenceData chainjson.Map `json:"reference_data"`
}

func (a *issueAction) Build(ctx context.Context, maxTime time.Time, builder *txbuilder.TemplateBuilder) error {
	if a.AssetID == (bc.AssetID{}) {
		return txbuilder.MissingFieldsError("asset_id")
	}

	asset, err := a.assets.findByID(ctx, a.AssetID)
	if errors.Root(err) == pg.ErrUserInputNotFound {
		err = errors.WithDetailf(err, "missing asset with ID %q", a.AssetID)
	}
	if err != nil {
		return err
	}

	var nonce [8]byte
	_, err = rand.Read(nonce[:])
	if err != nil {
		return err
	}
	txin := bc.NewIssuanceInput(nonce[:], a.Amount, a.ReferenceData, asset.InitialBlockHash, asset.IssuanceProgram, nil)

	tplIn := &txbuilder.SigningInstruction{AssetAmount: a.AssetAmount}
	path := signers.Path(asset.Signer, signers.AssetKeySpace)
	keyIDs := txbuilder.KeyIDs(asset.Signer.XPubs, path)
	tplIn.AddWitnessKeys(keyIDs, asset.Signer.Quorum)

	builder.RestrictMinTimeMS(bc.Millis(time.Now()))
	return builder.AddInput(txin, tplIn)
}
