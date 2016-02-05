package nodetxlog

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/fedchain/bc"
)

type NodeTx struct {
	ID       bc.Hash            `json:"id"`
	Time     time.Time          `json:"transaction_time"`
	Inputs   []nodeTxInput      `json:"inputs"`
	Outputs  []nodeTxOutput     `json:"outputs"`
	Metadata chainjson.HexBytes `json:"metadata"`
}

type nodeTxInput struct {
	Type            string             `json:"type"`
	TxID            *bc.Hash           `json:"transaction_id,omitempty"`
	TxOut           *uint32            `json:"transaction_output,omitempty"`
	AssetID         bc.AssetID         `json:"asset_id"`
	AssetLabel      string             `json:"asset_label,omitempty"`
	AssetDefinition chainjson.HexBytes `json:"asset_definition,omitempty"`
	Amount          uint64             `json:"amount"`
	Address         chainjson.HexBytes `json:"address,omitempty"`
	Script          chainjson.HexBytes `json:"script,omitempty"`
	AccountID       string             `json:"account_id,omitempty"`
	AccountLabel    string             `json:"account_label,omitempty"`
	Metadata        chainjson.HexBytes `json:"metadata"`

	mNodeID string
}

type nodeTxOutput struct {
	AssetID      bc.AssetID         `json:"asset_id"`
	AssetLabel   string             `json:"asset_label,omitempty"`
	Amount       uint64             `json:"amount"`
	Address      chainjson.HexBytes `json:"address,omitempty"`
	Script       chainjson.HexBytes `json:"script,omitempty"`
	AccountID    string             `json:"account_id,omitempty"`
	AccountLabel string             `json:"account_label,omitempty"`
	Metadata     chainjson.HexBytes `json:"metadata"`

	mNodeID string
}

// Write persists a transaction along with its metadata
// for every node (issuer, manager) associated with the transaction.
func Write(ctx context.Context, tx *bc.Tx, ts time.Time) error {
	ins, outs, err := appdb.GetActUTXOs(ctx, tx)
	if err != nil {
		return errors.Wrap(err, "fetching utxo account data")
	}

	assetMap := make(map[string]*appdb.ActAsset)
	var assetIDs []string
	for _, out := range tx.Outputs {
		assetIDs = append(assetIDs, out.AssetID.String())
	}
	assets, err := appdb.GetActAssets(ctx, assetIDs)
	if err != nil {
		return errors.Wrap(err, "getting assets")
	}
	for _, asset := range assets {
		assetMap[asset.ID] = asset
	}

	accountMap := make(map[string]*appdb.ActAccount)
	nodeAccounts := make(map[string][]string)
	var accountIDs []string
	for _, utxo := range append(ins, outs...) {
		if utxo.AccountID != "" {
			accountIDs = append(accountIDs, utxo.AccountID)
		}
	}
	accounts, err := appdb.GetActAccounts(ctx, accountIDs)
	if err != nil {
		return errors.Wrap(err, "getting accounts")
	}
	for _, acc := range accounts {
		accountMap[acc.ID] = acc
		nodeAccounts[acc.ManagerNodeID] = append(nodeAccounts[acc.ManagerNodeID], acc.ID)
	}

	NodeTx, err := generateNodeTx(tx, ins, outs, assetMap, accountMap, ts)
	if err != nil {
		return errors.Wrap(err, "generating master")
	}

	if tx.IsIssuance() {
		filteredTx, err := json.Marshal(filterAccounts(NodeTx, ""))
		if err != nil {
			return errors.Wrap(err, "filtering tx")
		}
		_, err = appdb.WriteIssuerTx(ctx, tx.Hash.String(), filteredTx, assets[0].IssuerNodeID, assets[0].ID)
		if err != nil {
			return errors.Wrap(err, "writing issuer tx")
		}
	}

	for nodeID, accountIDs := range nodeAccounts {
		filteredTx, err := json.Marshal(filterAccounts(NodeTx, nodeID))
		if err != nil {
			return errors.Wrap(err, "filtering tx")
		}
		_, err = appdb.WriteManagerTx(ctx, tx.Hash.String(), filteredTx, nodeID, accountIDs)
		if err != nil {
			return errors.Wrap(err, "writing manager tx")
		}
	}

	return nil
}

func generateNodeTx(
	tx *bc.Tx,
	ins []*appdb.ActUTXO,
	outs []*appdb.ActUTXO,
	assetMap map[string]*appdb.ActAsset,
	accountMap map[string]*appdb.ActAccount,
	ts time.Time,
) (*NodeTx, error) {
	actTx := &NodeTx{
		ID:       tx.Hash,
		Metadata: tx.Metadata,
		Time:     ts,
	}
	for i, in := range tx.Inputs {
		if in.IsIssuance() {
			var (
				amt     uint64
				assetID bc.AssetID
			)
			for _, o := range tx.Outputs {
				amt += o.Amount
				assetID = o.AssetID
			}
			asset := assetMap[assetID.String()]
			var label string
			if asset != nil {
				label = asset.Label
			}
			actTx.Inputs = append(actTx.Inputs, nodeTxInput{
				Type:            "issuance",
				AssetID:         assetID,
				AssetLabel:      label,
				AssetDefinition: in.AssetDefinition,
				Amount:          amt,
				Metadata:        in.Metadata,
			})
			continue
		}

		assetID, err := bc.ParseHash(ins[i].AssetID)
		if err != nil {
			return nil, errors.Wrap(err, "parsing utxo asset id")
		}
		asset := assetMap[ins[i].AssetID]
		var assetLabel string
		if asset != nil {
			assetLabel = asset.Label
		}

		account := accountMap[ins[i].AccountID]
		var accountLabel, mNodeID string
		if account != nil {
			accountLabel = account.Label
			mNodeID = account.ManagerNodeID
		}

		actTx.Inputs = append(actTx.Inputs, nodeTxInput{
			Type:         "transfer",
			TxID:         &in.Previous.Hash,
			TxOut:        &in.Previous.Index,
			AssetID:      bc.AssetID(assetID),
			AssetLabel:   assetLabel,
			Amount:       ins[i].Amount,
			AccountID:    ins[i].AccountID,
			AccountLabel: accountLabel,
			Address:      ins[i].Script,
			Script:       ins[i].Script,
			Metadata:     in.Metadata,
			mNodeID:      mNodeID,
		})
	}

	for i, out := range tx.Outputs {
		asset := assetMap[out.AssetID.String()]
		var assetLabel string
		if asset != nil {
			assetLabel = asset.Label
		}

		account := accountMap[outs[i].AccountID]
		var accountLabel, mNodeID string
		if account != nil {
			accountLabel = account.Label
			mNodeID = account.ManagerNodeID
		}

		actTx.Outputs = append(actTx.Outputs, nodeTxOutput{
			AssetID:      out.AssetID,
			AssetLabel:   assetLabel,
			Amount:       out.Amount,
			Address:      out.Script,
			Script:       out.Script,
			AccountID:    outs[i].AccountID,
			AccountLabel: accountLabel,
			Metadata:     out.Metadata,
			mNodeID:      mNodeID,
		})
	}
	return actTx, nil
}

func filterAccounts(tx *NodeTx, keepID string) *NodeTx {
	filteredTx := new(NodeTx)
	*filteredTx = *tx
	filteredTx.Inputs = make([]nodeTxInput, len(tx.Inputs))
	copy(filteredTx.Inputs, tx.Inputs)
	filteredTx.Outputs = make([]nodeTxOutput, len(tx.Outputs))
	copy(filteredTx.Outputs, tx.Outputs)
	for i := range filteredTx.Inputs {
		in := &filteredTx.Inputs[i]
		if in.mNodeID != keepID {
			in.AccountLabel = ""
			in.AccountID = ""
		}
	}
	for i := range filteredTx.Outputs {
		out := &filteredTx.Outputs[i]
		if out.mNodeID != keepID {
			out.AccountLabel = ""
			out.AccountID = ""
		}
	}
	return filteredTx
}
