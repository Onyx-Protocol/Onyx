package txbuilder

import (
	"context"

	"chain/core/pb"
	"chain/errors"
	"chain/protocol/bc"
)

func DecodeControlProgramAction(proto *pb.Action_ControlProgram) (Action, error) {
	assetID, err := bc.AssetIDFromBytes(proto.Asset.GetAssetId())
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return &controlProgramAction{
		AssetAmount:   bc.AssetAmount{AssetID: assetID, Amount: proto.Amount},
		Program:       proto.ControlProgram,
		ReferenceData: proto.ReferenceData,
	}, nil
}

type controlProgramAction struct {
	bc.AssetAmount
	Program       []byte
	ReferenceData []byte
}

func (a *controlProgramAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if len(a.Program) == 0 {
		missing = append(missing, "control_program")
	}
	if a.AssetID == (bc.AssetID{}) {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	out := bc.NewTxOutput(a.AssetID, a.Amount, a.Program, a.ReferenceData)
	return b.AddOutput(out)
}

func DecodeSetTxRefDataAction(proto *pb.Action_SetTxReferenceData) (Action, error) {
	return &setTxRefDataAction{Data: proto.Data}, nil
}

type setTxRefDataAction struct {
	Data []byte
}

func (a *setTxRefDataAction) Build(ctx context.Context, b *TemplateBuilder) error {
	if len(a.Data) == 0 {
		return MissingFieldsError("reference_data")
	}
	return b.setReferenceData(a.Data)
}
