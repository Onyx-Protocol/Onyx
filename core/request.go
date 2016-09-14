package core

import (
	"context"
	"encoding/json"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/txbuilder"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
)

var (
	errBadActionType = errors.New("bad action type")
	errBadAlias      = errors.New("bad alias")
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
		return errors.WithDetailf(errBadActionType, "unknown type %s", x.Type)
	}
	return json.Unmarshal(data, a.underlying)
}

type buildRequest struct {
	Tx            *bc.TxData    `json:"raw_transaction"`
	Actions       []*action     `json:"actions"`
	ReferenceData chainjson.Map `json:"reference_data"`
	TTL           time.Duration `json:"ttl"`
}

func (req *buildRequest) actions() []txbuilder.Action {
	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for _, act := range req.Actions {
		actions = append(actions, act.underlying)
	}

	return actions
}

func filterAliases(ctx context.Context, br *buildRequest) error {
	for i, aAction := range br.Actions {
		v := aAction.underlying
		switch p := v.(type) {
		case *asset.IssueAction:
			if (p.AssetID == bc.AssetID{}) && p.AssetAlias != "" {
				ast, err := asset.FindByAlias(ctx, p.AssetAlias)
				if err != nil {
					return errors.WithDetailf(err, "invalid asset alias %s on action %d", p.AssetAlias, i)
				}
				p.AssetID = ast.AssetID
			}
			aAction.underlying = p
		case *account.ControlAction:
			if (p.AssetID == bc.AssetID{}) && p.AssetAlias != "" {
				ast, err := asset.FindByAlias(ctx, p.AssetAlias)
				if err != nil {
					return errors.WithDetailf(err, "invalid asset alias %s on action %d", p.AssetAlias, i)
				}
				p.AssetID = ast.AssetID
			}
			if p.AccountID == "" && p.AccountAlias != "" {
				acc, err := account.FindByAlias(ctx, p.AccountAlias)
				if err != nil {
					return errors.WithDetailf(err, "invalid account alias %s on action %d", p.AccountAlias, i)
				}
				p.AccountID = acc.ID
			}
			aAction.underlying = p
		case *account.SpendAction:
			if (p.AssetID == bc.AssetID{}) && p.AssetAlias != "" {
				ast, err := asset.FindByAlias(ctx, p.AssetAlias)
				if err != nil {
					return errors.WithDetailf(err, "invalid asset alias %s on action %d", p.AssetAlias, i)
				}
				p.AssetID = ast.AssetID
			}
			if p.AccountID == "" && p.AccountAlias != "" {
				acc, err := account.FindByAlias(ctx, p.AccountAlias)
				if err != nil {
					return errors.WithDetailf(err, "invalid account alias %s on action %d", p.AccountAlias, i)
				}
				p.AccountID = acc.ID
			}
			aAction.underlying = p
		case *txbuilder.ControlProgramAction:
			if (p.AssetID == bc.AssetID{}) && p.AssetAlias != "" {
				ast, err := asset.FindByAlias(ctx, p.AssetAlias)
				if err != nil {
					return errors.WithDetailf(err, "invalid asset alias %s on action %d", p.AssetAlias, i)
				}
				p.AssetID = ast.AssetID
			}
			aAction.underlying = p
		}
	}
	return nil
}
