package core

import (
	"encoding/json"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/txbuilder"
	"chain/cos/bc"
	chainjson "chain/encoding/json"
	"chain/errors"
)

type action struct {
	underlying txbuilder.Action
}

func (a *action) UnmarshalJSON(data []byte) error {
	var x struct{ Type string }
	err := json.Unmarshal(data, &x)
	if err != nil {
		return err
	}

	switch x.Type {
	case "control_program":
		a.underlying = new(txbuilder.ControlProgramAction)
	case "spend_account_unspent_output_selector":
		a.underlying = new(account.SpendAction)
	case "account_control":
		a.underlying = new(account.ControlAction)
	case "issue":
		a.underlying = new(asset.IssueAction)
	default:
		return errors.WithDetailf(ErrBadBuildRequest, "invalid action: %s", x.Type)
	}
	return json.Unmarshal(data, a.underlying)
}

type buildRequest struct {
	Tx            *bc.TxData    `json:"transaction"`
	Actions       []*action     `json:"actions"`
	ReferenceData chainjson.Map `json:"reference_data"`
}

func (req *buildRequest) actions() []txbuilder.Action {
	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for _, act := range req.Actions {
		actions = append(actions, act.underlying)
	}

	return actions
}
