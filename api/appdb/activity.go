package appdb

import (
	"encoding/json"

	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/fedchain/bc"
)

func activityItemsFromRows(rows *sql.Rows) (items []*json.RawMessage, last string, err error) {
	for rows.Next() {
		var a []byte
		err := rows.Scan(&last, &a)
		if err != nil {
			err = errors.Wrap(err, "row scan")
			return nil, "", err
		}

		items = append(items, (*json.RawMessage)(&a))
	}

	if rows.Err() != nil {
		err = errors.Wrap(rows.Err(), "rows")
		return nil, "", err
	}

	return items, last, nil
}

type ActUTXO struct {
	AssetID       string
	Amount        uint64
	ManagerNodeID string
	AccountID     string
	Script        []byte
}

type ActAsset struct {
	ID           string
	Label        string
	IssuerNodeID string
	ProjID       string
}

type ActAccount struct {
	ID            string
	Label         string
	ManagerNodeID string
	ProjID        string
}

// GetActUTXOs returns information about outputs from both sides of a transaciton.
func GetActUTXOs(ctx context.Context, tx *bc.Tx) (ins, outs []*ActUTXO, err error) {
	var hashes []bc.Hash

	all := make(map[bc.Outpoint]*ActUTXO)
	scriptMap := make(map[string]bc.Outpoint)
	var scripts [][]byte

	for _, in := range tx.Inputs {
		if !in.IsIssuance() {
			hashes = append(hashes, in.Previous.Hash)
		}
	}

	for i, out := range tx.Outputs {
		all[bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}] = &ActUTXO{
			AssetID: out.AssetID.String(),
			Amount:  out.Amount,
			Script:  out.Script,
		}
		scriptMap[string(out.Script)] = bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}
		scripts = append(scripts, out.Script)
	}

	txs, err := txdb.GetTxs(ctx, hashes...) // modifies hashes
	if err != nil {
		return nil, nil, err
	}
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			continue
		}
		out := txs[in.Previous.Hash].Outputs[in.Previous.Index]
		all[in.Previous] = &ActUTXO{
			Amount:  out.Amount,
			AssetID: out.AssetID.String(),
			Script:  out.Script,
		}
		scriptMap[string(out.Script)] = in.Previous
		scripts = append(scripts, out.Script)
	}

	const scriptQ = `
		SELECT pk_script, account_id, manager_node_id FROM addresses WHERE pk_script=ANY($1)
	`
	rows, err := pg.Query(ctx, scriptQ, pg.Byteas(scripts))
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var (
			script             []byte
			accountID, mNodeID string
		)
		err := rows.Scan(&script, &accountID, &mNodeID)
		if err != nil {
			return nil, nil, err
		}
		utxo := all[scriptMap[string(script)]]
		utxo.AccountID = accountID
		utxo.ManagerNodeID = mNodeID
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	for _, in := range tx.Inputs {
		ins = append(ins, all[in.Previous]) // nil for issuance
	}

	for i := range tx.Outputs {
		op := bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}
		outs = append(outs, all[op])
	}

	return ins, outs, nil
}

func GetActAssets(ctx context.Context, assetIDs []string) ([]*ActAsset, error) {
	q := `
		SELECT a.id, a.label, i.id, i.project_id
		FROM assets a
		JOIN issuer_nodes i ON a.issuer_node_id = i.id
		WHERE a.id = ANY($1)
	`
	rows, err := pg.Query(ctx, q, pg.Strings(assetIDs))
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var res []*ActAsset
	for rows.Next() {
		a := new(ActAsset)
		err := rows.Scan(&a.ID, &a.Label, &a.IssuerNodeID, &a.ProjID)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		res = append(res, a)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return res, nil
}

func GetActAccounts(ctx context.Context, accountIDs []string) ([]*ActAccount, error) {
	q := `
		SELECT acc.id, acc.label, acc.manager_node_id, mn.project_id
		FROM accounts acc
		JOIN manager_nodes mn ON acc.manager_node_id = mn.id
		WHERE acc.id = ANY($1)
		ORDER BY acc.id
	`
	rows, err := pg.Query(ctx, q, pg.Strings(accountIDs))
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var res []*ActAccount
	for rows.Next() {
		a := new(ActAccount)
		err := rows.Scan(&a.ID, &a.Label, &a.ManagerNodeID, &a.ProjID)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		res = append(res, a)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return res, nil
}

// TODO(jeffomatic): This is identical to asset.isIssuance, but is copied here
// to avoid circular dependencies betwen the two packages. This should probably
// be moved to the fedchain(-sandbox?)/wire package at some point.
func isIssuance(msg *bc.Tx) bool {
	if len(msg.Inputs) == 1 && msg.Inputs[0].IsIssuance() {
		if len(msg.Outputs) == 0 {
			return false
		}
		assetID := msg.Outputs[0].AssetID
		for _, out := range msg.Outputs {
			if out.AssetID != assetID {
				return false
			}
		}
		return true
	}
	return false
}
