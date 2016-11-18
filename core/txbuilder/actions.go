package txbuilder

import (
	"context"
	stdjson "encoding/json"
	"time"

	"chain/encoding/json"
	"chain/protocol/bc"
)

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

func (c *controlProgramAction) Build(ctx context.Context, maxTime time.Time, b *TemplateBuilder) error {
	out := bc.NewTxOutput(c.AssetID, c.Amount, c.Program, c.ReferenceData)
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

func (a *setTxRefDataAction) Build(ctx context.Context, maxTime time.Time, b *TemplateBuilder) error {
	return b.setReferenceData(a.Data)
}
