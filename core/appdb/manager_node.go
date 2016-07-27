package appdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/database/pg"
	"chain/errors"
)

// NodeKey is represents a single key in a node's multi-sig configuration.
// It is used as a return value when nodes are created.
//
// A NodeKey consists of a type, plus parameters depending on that type. Valid
// manager node types include "node" and "account". For issuer nodes, only
// "node" is valid.
//
// For node-type keys, XPrv will be populated only if it was generated
// server-side when the node was created.
type NodeKey struct {
	Type string `json:"type"`

	// Parameters for type "node"
	XPub *hdkey.XKey `json:"xpub,omitempty"`
	XPrv *hdkey.XKey `json:"xprv,omitempty"`
}

func buildNodeKeys(xpubs, xprvs []*hdkey.XKey) ([]*NodeKey, error) {
	pubToPrv := make(map[string]*hdkey.XKey)
	for i, xprv := range xprvs {
		xpub, err := xprv.Neuter()
		if err != nil {
			return nil, errors.Wrapf(err, "cannot extract xpub from xprv: %d", i)
		}

		k := &hdkey.XKey{ExtendedKey: *xpub}
		pubToPrv[k.String()] = xprv
	}

	var res []*NodeKey
	for _, xpub := range xpubs {
		k := &NodeKey{Type: "service", XPub: xpub}

		s := xpub.String()
		if xprv := pubToPrv[s]; xprv != nil {
			k.XPrv = xprv
		}

		res = append(res, k)
	}

	return res, nil
}

// ManagerNode represents a single manager node. It is intended to be used wth API
// responses.
type ManagerNode struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	Keys     []*NodeKey `json:"keys"`
	SigsReqd int        `json:"signatures_required"`
}

// InsertManagerNode inserts a new manager node into the database. If a manager node
// already exists with the provided project ID and client token, this function will
// return the existing manager node.
func InsertManagerNode(ctx context.Context, projID, label string, xpubs, gennedKeys []*hdkey.XKey, variableKeys, sigsRequired int, clientToken *string) (w *ManagerNode, err error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction
	const q = `
		INSERT INTO manager_nodes (label, project_id, generated_keys, variable_keys, sigs_required, client_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (project_id, client_token) DO NOTHING
		RETURNING id
	`
	var id string
	xprvs := keysToStrings(gennedKeys)
	err = pg.QueryRow(ctx, q, label, projID, pg.Strings(xprvs), variableKeys, sigsRequired, clientToken).Scan(&id)
	if err == sql.ErrNoRows && clientToken != nil {
		// A sql.ErrNoRows error here indicates that we failed to insert
		// a manager node because there was a conflict on the client token.
		// A previous request to create this manager node succeeded.
		return nil, errors.Wrap(err, "looking up existing account manager")
	}
	if err != nil {
		return nil, errors.Wrap(err, "insert account manager")
	}

	err = createRotation(ctx, id, keysToStrings(xpubs)...)
	if err != nil {
		return nil, errors.Wrap(err, "create rotation")
	}

	keys, err := buildNodeKeys(xpubs, gennedKeys)
	if err != nil {
		return nil, errors.Wrap(err, "generating account manager key list")
	}

	for i := 0; i < variableKeys; i++ {
		keys = append(keys, &NodeKey{Type: "account"})
	}

	return &ManagerNode{
		ID:       id,
		Label:    label,
		Keys:     keys,
		SigsReqd: sigsRequired,
	}, nil
}

// Balance is a struct describing the balance of
// an asset that a manager node or account has.
type Balance struct {
	AssetID   bc.AssetID `json:"asset_id"`
	Confirmed int64      `json:"confirmed"`
	Total     int64      `json:"total"`
}

// AccountBalanceItem is returned by AccountsWithAsset
type AccountBalanceItem struct {
	AccountID string `json:"account_id"`
	Confirmed int64  `json:"confirmed"`
	Total     int64  `json:"total"`
}

// AccountsWithAsset fetches the balance of a particular asset
// within a manager node, grouped and sorted by individual accounts.
//
// EXPERIMENTAL - implemented for Glitterco
func AccountsWithAsset(ctx context.Context, mnodeID, assetID, prev string, limit int) ([]*AccountBalanceItem, string, error) {
	const q = `
		WITH combined_utxos AS (
			SELECT a.amount, a.asset_id, a.tx_hash, a.index,
			manager_node_id, account_id,
			confirmed_in IS NOT NULL as confirmed,
			reservation_id IS NOT NULL as spent_in_pool
			FROM account_utxos a
			WHERE manager_node_id=$1 AND a.asset_id=$2 AND ($3='' OR account_id>$3)
		), amounts AS (
			SELECT
				(CASE WHEN confirmed THEN amount ELSE 0 END) as confirmed_amount,
				(CASE WHEN NOT spent_in_pool THEN amount ELSE 0 END) as total_amount,
				account_id FROM combined_utxos
				WHERE confirmed OR NOT spent_in_pool
		)

		SELECT sum(confirmed_amount), sum(total_amount), account_id
		FROM amounts
		JOIN accounts ON accounts.id = account_id
		WHERE NOT accounts.archived
		GROUP BY account_id
		ORDER BY account_id ASC
		LIMIT $4
	`
	rows, err := pg.Query(ctx, q, mnodeID, assetID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "balances query")
	}
	defer rows.Close()

	var (
		bals []*AccountBalanceItem
		last string
	)
	for rows.Next() {
		var item AccountBalanceItem
		err = rows.Scan(&item.Confirmed, &item.Total, &item.AccountID)
		if err != nil {
			return nil, "", errors.Wrap(err, "rows scan")
		}
		bals = append(bals, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "rows error")
	}

	if len(bals) > 0 {
		last = bals[len(bals)-1].AccountID
	}

	return bals, last, nil
}

func createRotation(ctx context.Context, managerNodeID string, xpubs ...string) error {
	const q = `
		WITH new_rotation AS (
			INSERT INTO rotations (manager_node_id, keyset)
			VALUES ($1, $2)
			RETURNING id
		)
		UPDATE manager_nodes SET current_rotation=(SELECT id FROM new_rotation)
		WHERE id=$1
	`
	_, err := pg.Exec(ctx, q, managerNodeID, pg.Strings(xpubs))
	return err
}

func managerNodeVariableKeys(ctx context.Context, managerNodeID string) (int, error) {
	const q = `SELECT variable_keys FROM manager_nodes WHERE id = $1`
	count := 0
	err := pg.QueryRow(ctx, q, managerNodeID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
