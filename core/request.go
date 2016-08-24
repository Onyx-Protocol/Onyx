package core

import (
	"context"
	"encoding/json"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/txbuilder"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
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
	case "control_account":
		a.underlying = new(account.ControlAction)
	case "issue":
		a.underlying = new(asset.IssueAction)
	case "spend_account_unspent_output":
		a.underlying = new(account.SpendUTXOAction)
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

// aliasBuildRequest is like a buildRequest, but includes aliases for accounts
// and assets. The aliases can be used to populate the account id or asset id
// field and then this can be turned into a buildRequest. The Tx and ReferenceData
// fields are json.RawMessages because we don't need them when doing alias lookups.
type aliasBuildRequest struct {
	Tx            *json.RawMessage         `json:"transaction"`
	Actions       []map[string]interface{} `json:"actions"`
	ReferenceData *json.RawMessage         `json:"reference_data"`
}

func filterAliases(ctx context.Context, abr *aliasBuildRequest) (*buildRequest, error) {
	// parse aliases as needed
	var err error
	for _, aAction := range abr.Actions {
		p0, ok := aAction["params"]
		if !ok {
			return nil, errors.Wrap(ErrBadBuildRequest, "missing params on action")
		}

		p, ok := p0.(map[string]interface{})
		if !ok {
			return nil, errors.Wrap(ErrBadBuildRequest, "misshappen params on action")
		}

		if _, ok := p["asset_id"]; !ok {
			if assetAlias, ok := p["asset_alias"]; ok {
				aa, ok := assetAlias.(string)
				if !ok {
					return nil, errors.Wrap(ErrBadBuildRequest, "misshappen asset alias")
				}

				ast, err := asset.FindByAlias(ctx, aa)
				if err != nil {
					return nil, errors.Wrap(ErrBadBuildRequest, "missing asset alias")
				}

				p["asset_id"] = ast.AssetID
			}
		}

		if _, ok := p["account_id"]; !ok {
			if accountAlias, ok := p["account_alias"]; ok {
				aa, ok := accountAlias.(string)
				if !ok {
					return nil, errors.Wrap(ErrBadBuildRequest, "misshappen account alias")
				}

				acc, err := account.FindByAlias(ctx, aa)
				if err != nil {
					return nil, errors.Wrap(ErrBadBuildRequest, "missing account alias")
				}

				p["account_id"] = acc.ID
			}
		}
	}

	// turn aliasBuildRequest into buildRequest
	b, err := json.Marshal(abr)
	if err != nil {
		return nil, err
	}

	var br buildRequest
	err = json.Unmarshal(b, &br)
	if err != nil {
		return nil, err
	}

	return &br, nil
}
