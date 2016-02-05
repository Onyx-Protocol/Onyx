package appdb

import (
	"encoding/json"
	"fmt"

	"golang.org/x/net/context"

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
	var (
		txHashStr  = tx.Hash.String()
		isIssuance = tx.IsIssuance()

		hashes  []string
		indexes []uint32
	)

	if !isIssuance {
		for _, in := range tx.Inputs {
			hashes = append(hashes, in.Previous.Hash.String())
			indexes = append(indexes, in.Previous.Index)
		}
	}

	for i := range tx.Outputs {
		hashes = append(hashes, txHashStr)
		indexes = append(indexes, uint32(i))
	}

	// Both confirmed (blockchain) utxos and unconfirmed (pool) utxos
	const q = `
		WITH outpoints AS (SELECT unnest($1::text[]), unnest($2::bigint[]))
			SELECT tx_hash, index,
				asset_id, amount, script,
				account_id, manager_node_id
			FROM utxos
			WHERE (tx_hash, index) IN (TABLE outpoints)
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, pg.Strings(hashes), pg.Uint32s(indexes))
	if err != nil {
		return nil, nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	all := make(map[bc.Outpoint]*ActUTXO)
	for rows.Next() {
		var (
			hash  bc.Hash
			index uint32
			utxo  = new(ActUTXO)
		)
		err := rows.Scan(
			&hash, &index,
			&utxo.AssetID, &utxo.Amount, &utxo.Script,
			&utxo.AccountID, &utxo.ManagerNodeID,
		)
		if err != nil {
			return nil, nil, errors.Wrap(err, "row scan")
		}

		all[bc.Outpoint{Hash: hash, Index: index}] = utxo
	}
	if rows.Err() != nil {
		return nil, nil, errors.Wrap(rows.Err(), "end row scan loop")
	}

	if len(all) != len(hashes) {
		err := fmt.Errorf("found %d utxos for %d outpoints", len(all), len(hashes))
		return nil, nil, errors.Wrap(err)
	}

	if !isIssuance {
		for _, in := range tx.Inputs {
			ins = append(ins, all[in.Previous])
		}
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
		ORDER BY a.id
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, pg.Strings(assetIDs))
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
	rows, err := pg.FromContext(ctx).Query(ctx, q, pg.Strings(accountIDs))
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
