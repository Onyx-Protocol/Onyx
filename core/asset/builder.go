package asset

import (
	"context"
	"crypto/rand"
	"time"

	"chain/core/pb"
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

func (reg *Registry) DecodeIssueAction(proto *pb.Action_Issue) (txbuilder.Action, error) {
	assetID, err := bc.AssetIDFromBytes(proto.Asset.GetAssetId())
	if err != nil {
		return nil, errors.Wrap(err)
	}
	a := &issueAction{
		assets:        reg,
		AssetAmount:   bc.AssetAmount{AssetID: assetID, Amount: proto.Amount},
		ReferenceData: proto.ReferenceData,
	}
	return a, nil
}

type issueAction struct {
	assets *Registry
	bc.AssetAmount
	ReferenceData []byte
}

func (a *issueAction) Build(ctx context.Context, builder *txbuilder.TemplateBuilder) error {
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

	assetdef := asset.RawDefinition()

	txin := bc.NewIssuanceInput(nonce[:], a.Amount, a.ReferenceData, asset.InitialBlockHash, asset.IssuanceProgram, nil, assetdef)

	tplIn := &pb.TxTemplate_SigningInstruction{AssetId: a.AssetID[:], Amount: a.Amount}
	path := signers.Path(asset.Signer, signers.AssetKeySpace)
	tplIn.WitnessComponents = append(tplIn.WitnessComponents, pb.SignatureWitness(asset.Signer.XPubs, path, asset.Signer.Quorum))

	builder.RestrictMinTime(time.Now())
	return builder.AddInput(txin, tplIn)
}
