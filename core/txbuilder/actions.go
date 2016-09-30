package txbuilder

import (
	"context"

	"chain/encoding/json"
	"chain/protocol/bc"
)

type ControlProgramAction struct {
	bc.AssetAmount
	Program json.HexBytes `json:"control_program"`

	// This field is only necessary for filtering
	// aliases on transaction build requests. A wrapper
	// function reads it to set the ID field. It is
	// not used anywhere else in the code base.
	AssetAlias string `json:"asset_alias"`

	ReferenceData json.Map `json:"reference_data"`
}

func (c *ControlProgramAction) Build(ctx context.Context) (*BuildResult, error) {
	out := bc.NewTxOutput(c.AssetID, c.Amount, c.Program, c.ReferenceData)
	return &BuildResult{Outputs: []*bc.TxOutput{out}}, nil
}

type SetTxRefDataAction struct {
	Data json.Map `json:"reference_data"`
}

func (a *SetTxRefDataAction) Build(ctx context.Context) (*BuildResult, error) {
	return &BuildResult{ReferenceData: a.Data}, nil
}
