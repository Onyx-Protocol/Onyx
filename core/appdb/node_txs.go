package appdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

// WriteIssuerTx records an issuance tx for this issuer to the database.
func WriteIssuerTx(ctx context.Context, txHash string, data []byte, iNodeID string, ts time.Time, assetIDs []string) (id string, err error) {
	const issuerQ = `
		INSERT INTO issuer_txs (issuer_node_id, tx_hash, data, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (issuer_node_id, tx_hash) DO UPDATE SET data = excluded.data
		RETURNING id
	`
	err = pg.QueryRow(ctx, issuerQ, iNodeID, txHash, data, ts).Scan(&id)
	if err != nil {
		return "", errors.Wrap(err, "insert issuer tx")
	}

	const assetQ = `
		INSERT INTO issuer_txs_assets (issuer_tx_id, asset_id)
		VALUES ($1, unnest($2::text[]))
		ON CONFLICT (issuer_tx_id, asset_id) DO NOTHING
	`
	_, err = pg.Exec(ctx, assetQ, id, pg.Strings(assetIDs))
	return id, errors.Wrap(err, "insert issuer tx for assets")
}

// WriteManagerTx records a transaction for this account manager to the
// database.
func WriteManagerTx(ctx context.Context, txHash string, data []byte, mNodeID string, ts time.Time, accounts []string) (id string, err error) {
	const managerQ = `
		INSERT INTO manager_txs (manager_node_id, tx_hash, data, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (manager_node_id, tx_hash) DO UPDATE SET data = excluded.data
		RETURNING id
	`
	err = pg.QueryRow(ctx, managerQ, mNodeID, txHash, data, ts).Scan(&id)
	if err != nil {
		return "", errors.Wrap(err, "insert manager tx")
	}

	const accountQ = `
		INSERT INTO manager_txs_accounts (manager_tx_id, account_id)
		VALUES ($1, unnest($2::text[]))
		ON CONFLICT (manager_tx_id, account_id) DO NOTHING
	`
	_, err = pg.Exec(ctx, accountQ, id, pg.Strings(accounts))
	return id, errors.Wrap(err, "insert manager tx for account")
}

func ManagerTxs(ctx context.Context, managerNodeID string, prev string, limit int) ([]*json.RawMessage, string, error) {
	q := `
		SELECT id, data FROM manager_txs
		WHERE manager_node_id=$1 AND (($2 = '') OR (id < $2))
		ORDER BY id DESC LIMIT $3
	`

	rows, err := pg.Query(ctx, q, managerNodeID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "query")
	}
	defer rows.Close()

	return activityItemsFromRows(rows)
}

func AccountTxs(ctx context.Context, accountID string, startTime, endTime time.Time, prev string, limit int) ([]*json.RawMessage, string, error) {
	q := `
		SELECT mt.id, mt.data
		FROM manager_txs AS mt
		LEFT JOIN manager_txs_accounts AS a
		ON mt.id=a.manager_tx_id
		WHERE a.account_id=$1 AND (($2 = '') OR (mt.id < $2))
			AND mt.created_at >= $3 AND mt.created_at <= $4
		ORDER BY mt.id DESC
	`

	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := pg.Query(ctx, q, accountID, prev, startTime, endTime)
	if err != nil {
		return nil, "", errors.Wrap(err, "query")
	}
	defer rows.Close()

	return activityItemsFromRows(rows)
}

func IssuerTxs(ctx context.Context, inodeID string, prev string, limit int) ([]*json.RawMessage, string, error) {
	q := `
		SELECT id, data FROM issuer_txs
		WHERE issuer_node_id = $1 AND (($2 = '') OR (id < $2))
		ORDER BY id DESC LIMIT $3
	`
	rows, err := pg.Query(ctx, q, inodeID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "query")
	}
	defer rows.Close()

	return activityItemsFromRows(rows)
}

func AssetTxs(ctx context.Context, assetID string, prev string, limit int) ([]*json.RawMessage, string, error) {
	q := `
		SELECT it.id, it.data
		FROM issuer_txs AS it
		LEFT JOIN issuer_txs_assets AS a
		ON it.id = a.issuer_tx_id
		WHERE a.asset_id = $1 AND (($2 = '') OR (it.id < $2))
		ORDER BY it.id DESC LIMIT $3
	`
	rows, err := pg.Query(ctx, q, assetID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "query")
	}
	defer rows.Close()

	return activityItemsFromRows(rows)
}

func ManagerTx(ctx context.Context, managerNodeID, txID string) (*json.RawMessage, error) {
	q := `
		SELECT data FROM manager_txs
		WHERE manager_node_id=$1 AND tx_hash=$2
	`

	var a []byte
	err := pg.QueryRow(ctx, q, managerNodeID, txID).Scan(&a)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "transaction id: %v", txID)
	}
	return (*json.RawMessage)(&a), err
}
