package txbuilder

import (
	"context"
	stdjson "encoding/json"

	"chain/encoding/json"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

var retirementProgram = vmutil.NewBuilder().AddOp(vm.OP_FAIL).Program

func DecodeControlWithReceiverAction(data []byte) (Action, error) {
	a := new(controlWithReceiverAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type controlWithReceiverAction struct {
	bc.AssetAmount
	Receiver      *Receiver `json:"receiver"`
	ReferenceData json.Map  `json:"reference_data"`
}

func (a *controlWithReceiverAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.Receiver == nil {
		missing = append(missing, "receiver")
	} else {
		if len(a.Receiver.ControlProgram) == 0 {
			missing = append(missing, "receiver.control_program")
		}
		if a.Receiver.ExpiresAt.IsZero() {
			missing = append(missing, "receiver.expires_at")
		}
	}
	if a.AssetID == (bc.AssetID{}) {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	b.RestrictMaxTime(a.Receiver.ExpiresAt)
	out := bc.NewTxOutput(a.AssetID, a.Amount, a.Receiver.ControlProgram, a.ReferenceData)
	return b.AddOutput(out)
}

func DecodeControlProgramAction(data []byte) (Action, error) {
	a := new(controlProgramAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type controlProgramAction struct {
	bc.AssetAmount
	Program       json.HexBytes `json:"control_program"`
	ReferenceData json.Map      `json:"reference_data"`
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

func DecodeSetTxRefDataAction(data []byte) (Action, error) {
	a := new(setTxRefDataAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type setTxRefDataAction struct {
	Data json.Map `json:"reference_data"`
}

func (a *setTxRefDataAction) Build(ctx context.Context, b *TemplateBuilder) error {
	if len(a.Data) == 0 {
		return MissingFieldsError("reference_data")
	}
	return b.setReferenceData(a.Data)
}

func DecodeRetireAction(data []byte) (Action, error) {
	a := new(retireAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type retireAction struct {
	bc.AssetAmount
	ReferenceData json.Map `json:"reference_data"`
}

func (a *retireAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.AssetID == (bc.AssetID{}) {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	out := bc.NewTxOutput(a.AssetID, a.Amount, retirementProgram, a.ReferenceData)
	return b.AddOutput(out)
}
