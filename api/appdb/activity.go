package appdb

import (
	"encoding/json"

	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
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
	err = pg.ForQueryRows(ctx, scriptQ, pg.Byteas(scripts), func(script []byte, accountID, mnodeID string) {
		utxo := all[scriptMap[string(script)]]
		utxo.AccountID = accountID
		utxo.ManagerNodeID = mnodeID
	})
	if err != nil {
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
	var res []*ActAsset
	err := pg.ForQueryRows(ctx, q, pg.Strings(assetIDs), func(id, label, inodeID, projID string) {
		res = append(res, &ActAsset{ID: id, Label: label, IssuerNodeID: inodeID, ProjID: projID})
	})
	return res, err
}

func GetActAccounts(ctx context.Context, accountIDs []string) ([]*ActAccount, error) {
	q := `
		SELECT acc.id, acc.label, acc.manager_node_id, mn.project_id
		FROM accounts acc
		JOIN manager_nodes mn ON acc.manager_node_id = mn.id
		WHERE acc.id = ANY($1)
		ORDER BY acc.id
	`
	var res []*ActAccount
	err := pg.ForQueryRows(ctx, q, pg.Strings(accountIDs), func(id, label, mnodeID, projID string) {
		res = append(res, &ActAccount{ID: id, Label: label, ManagerNodeID: mnodeID, ProjID: projID})
	})
	return res, err
}
