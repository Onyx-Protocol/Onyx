package txbuilder

import (
	"chain/cos/bc"
	"chain/encoding/json"

	"golang.org/x/net/context"
)

type ControlProgramAction struct {
	Params struct {
		bc.AssetAmount
		Program json.HexBytes `json:"control_program"`
	}
	ReferenceData json.HexBytes `json:"reference_data"`
}

func (c *ControlProgramAction) Build(ctx context.Context) ([]*bc.TxInput, []*bc.TxOutput, []*Input, error) {
	out := bc.NewTxOutput(c.Params.AssetID, c.Params.Amount, c.Params.Program, c.ReferenceData)
	return nil, []*bc.TxOutput{out}, nil, nil
}
