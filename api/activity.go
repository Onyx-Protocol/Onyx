package api

import (
	"chain/api/asset/nodetxlog"
	chainjson "chain/encoding/json"
	"chain/fedchain/bc"
	"encoding/json"
)

type actEntry struct {
	Address      chainjson.HexBytes `json:"address,omitempty"`
	AccountID    string             `json:"account_id,omitempty"`
	AccountLabel string             `json:"account_label,omitempty"`

	Amount     int64  `json:"amount"`
	AssetID    string `json:"asset_id"`
	AssetLabel string `json:"asset_label"`
}

func nodeKey(assetID bc.AssetID, accountID string, address []byte) string {
	if accountID != "" {
		return assetID.String() + accountID
	}
	return assetID.String() + string(address)
}

func nodeTxsToActivity(acts []*json.RawMessage) ([]interface{}, error) {
	var results []interface{}
	for _, act := range acts {
		res, err := nodeTxToActivity(*act)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

func nodeTxToActivity(act []byte) (interface{}, error) {
	var nodeTx nodetxlog.NodeTx
	err := json.Unmarshal(act, &nodeTx)
	if err != nil {
		return nil, err
	}
	entries := make(map[string]*actEntry)
	for _, in := range nodeTx.Inputs {
		if in.Type == "issuance" {
			continue
		}
		key := nodeKey(in.AssetID, in.AccountID, in.Address)
		entry, ok := entries[key]
		if !ok {
			entry = &actEntry{
				AccountID:    in.AccountID,
				AccountLabel: in.AccountLabel,
				AssetID:      in.AssetID.String(),
				AssetLabel:   in.AssetLabel,
				Address:      in.Address,
				Amount:       0,
			}
			entries[key] = entry
		}

		entry.Amount += int64(in.Amount)
	}
	for _, out := range nodeTx.Outputs {
		key := nodeKey(out.AssetID, out.AccountID, out.Address)
		entry, ok := entries[key]
		if !ok {
			entry = &actEntry{
				AccountID:    out.AccountID,
				AccountLabel: out.AccountLabel,
				AssetID:      out.AssetID.String(),
				AssetLabel:   out.AssetLabel,
				Address:      out.Address,
				Amount:       0,
			}
			entries[key] = entry
		}
		entry.Amount -= int64(out.Amount)
	}
	var ins, outs []*actEntry
	for _, entry := range entries {
		if entry.Amount == 0 {
			continue
		}
		if entry.Amount < 0 {
			outs = append(outs, entry)
			entry.Amount *= -1
			continue
		}
		ins = append(ins, entry)
	}

	return map[string]interface{}{
		"transaction_id":   nodeTx.ID,
		"transaction_time": nodeTx.Time,
		"inputs":           ins,
		"outputs":          outs,
	}, nil
}
