package core

import (
	"context"

	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
)

var (
	errBadActionType = errors.New("bad action type")
	errBadAlias      = errors.New("bad alias")
	errBadAction     = errors.New("bad action object")
)

type buildRequest struct {
	Tx      *bc.TxData               `json:"base_transaction"`
	Actions []map[string]interface{} `json:"actions"`
	TTL     json.Duration            `json:"ttl"`
}

func (h *Handler) filterAliases(ctx context.Context, br *buildRequest) error {
	for i, m := range br.Actions {
		id, _ := m["assset_id"].(string)
		alias, _ := m["asset_alias"].(string)
		if id == "" && alias != "" {
			asset, err := h.Assets.FindByAlias(ctx, alias)
			if err != nil {
				return errors.WithDetailf(err, "invalid asset alias %s on action %d", alias, i)
			}
			m["asset_id"] = asset.AssetID
		}

		id, _ = m["account_id"].(string)
		alias, _ = m["account_alias"].(string)
		if id == "" && alias != "" {
			acc, err := h.Accounts.FindByAlias(ctx, alias)
			if err != nil {
				return errors.WithDetailf(err, "invalid account alias %s on action %d", alias, i)
			}
			m["account_id"] = acc.ID
		}
	}
	return nil
}
