package appdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
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
		k := &NodeKey{Type: "node", XPub: xpub}

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

// InsertManagerNode inserts a new manager node into the database.
func InsertManagerNode(ctx context.Context, projID, label string, xpubs, gennedKeys []*hdkey.XKey, variableKeys, sigsRequired int) (w *ManagerNode, err error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction
	const q = `
		INSERT INTO manager_nodes (label, project_id, generated_keys, variable_keys, sigs_required)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id string
	xprvs := keysToStrings(gennedKeys)
	err = pg.FromContext(ctx).QueryRow(ctx, q, label, projID, pg.Strings(xprvs), variableKeys, sigsRequired).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "insert manager node")
	}

	err = createRotation(ctx, id, keysToStrings(xpubs)...)
	if err != nil {
		return nil, errors.Wrap(err, "create rotation")
	}

	keys, err := buildNodeKeys(xpubs, gennedKeys)
	if err != nil {
		return nil, errors.Wrap(err, "generating node key list")
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

// GetManagerNode returns basic information about a single manager node.
func GetManagerNode(ctx context.Context, managerNodeID string) (*ManagerNode, error) {
	var (
		q = `
			SELECT label, keyset, generated_keys, variable_keys, sigs_required
			FROM manager_nodes mn
			JOIN rotations r ON r.id=mn.current_rotation
			WHERE mn.id = $1
		`
		label       string
		pubKeyStrs  []string
		privKeyStrs []string
		varKeys     int
		sigsReqd    int
	)
	err := pg.FromContext(ctx).QueryRow(ctx, q, managerNodeID).Scan(
		&label,
		(*pg.Strings)(&pubKeyStrs),
		(*pg.Strings)(&privKeyStrs),
		&varKeys,
		&sigsReqd,
	)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "manager node ID: %v", managerNodeID)
	}
	if err != nil {
		return nil, err
	}

	xpubs, err := stringsToKeys(pubKeyStrs)
	if err != nil {
		return nil, errors.Wrap(err, "parsing pub keys")
	}

	xprvs, err := stringsToKeys(privKeyStrs)
	if err != nil {
		return nil, errors.Wrap(err, "parsing private keys")
	}

	keys, err := buildNodeKeys(xpubs, xprvs)
	if err != nil {
		return nil, errors.Wrap(err, "generating node key list")
	}

	for i := 0; i < varKeys; i++ {
		keys = append(keys, &NodeKey{Type: "account"})
	}

	return &ManagerNode{
		ID:       managerNodeID,
		Label:    label,
		Keys:     keys,
		SigsReqd: sigsReqd,
	}, nil
}

// EXPERIMENTAL - implemented for Glitterco
//
// AccountsWithAsset fetches the balance of a particular asset
// within a manager node, grouped and sorted by individual accounts.
func AccountsWithAsset(ctx context.Context, mnodeID, assetID, prev string, limit int) ([]*AccountBalanceItem, string, error) {
	const q = `
		SELECT SUM(confirmed), SUM(unconfirmed), account_id
		FROM (
			SELECT amount AS confirmed, 0 AS unconfirmed, account_id
				FROM utxos WHERE confirmed AND manager_node_id=$1 AND asset_id=$2
					AND ($3='' OR account_id>$3)
			UNION ALL
			SELECT 0 AS confirmed, amount AS unconfirmed, account_id
				FROM utxos po WHERE NOT po.confirmed AND manager_node_id=$1 AND asset_id=$2
					AND ($3='' OR account_id>$3)
				AND NOT EXISTS(
					SELECT 1 FROM pool_inputs pi
					WHERE po.tx_hash = pi.tx_hash AND po.index = pi.index
				)
			UNION ALL
			SELECT 0 AS confirmed, amount*-1 AS unconfirmed, account_id
				FROM utxos u WHERE u.confirmed AND manager_node_id=$1 AND asset_id=$2
					AND ($3='' OR account_id>$3)
				AND EXISTS(
					SELECT 1 FROM pool_inputs pi
					WHERE u.tx_hash = pi.tx_hash AND u.index = pi.index
				)
		) AS bals
		INNER JOIN accounts ON accounts.id = account_id
		WHERE NOT accounts.archived
		GROUP BY account_id
		ORDER BY account_id ASC
		LIMIT $4
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, mnodeID, assetID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "balances query")
	}
	defer rows.Close()

	var (
		bals []*AccountBalanceItem
		last string
	)
	for rows.Next() {
		var (
			accountID    string
			conf, unconf int64
		)
		err = rows.Scan(&conf, &unconf, &accountID)
		if err != nil {
			return nil, "", errors.Wrap(err, "rows scan")
		}
		bals = append(bals, &AccountBalanceItem{accountID, conf, conf + unconf})
	}
	if err := rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "rows error")
	}

	if len(bals) > 0 {
		last = bals[len(bals)-1].AccountID
	}

	return bals, last, nil
}

// ListManagerNodes returns a list of active manager nodes contained in the given project.
func ListManagerNodes(ctx context.Context, projID string) ([]*ManagerNode, error) {
	q := `
		SELECT id, label
		FROM manager_nodes
		WHERE project_id = $1 AND NOT archived
		ORDER BY id
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, projID)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var managerNodes []*ManagerNode
	for rows.Next() {
		m := new(ManagerNode)
		err := rows.Scan(&m.ID, &m.Label)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		managerNodes = append(managerNodes, m)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return managerNodes, nil
}

// UpdateManagerNode updates the label of a manager node.
func UpdateManagerNode(ctx context.Context, mnodeID string, label *string) error {
	if label == nil {
		return nil
	}
	const q = `UPDATE manager_nodes SET label = $2 WHERE id = $1`
	db := pg.FromContext(ctx)
	_, err := db.Exec(ctx, q, mnodeID, *label)
	return errors.Wrap(err, "update query")
}

// ArchiveManagerNode marks a manager node as archived.
// Archived manager nodes do not appear for their parent projects,
// in the dashboard or for listManagerNodes. They cannot create new
// accounts or initiate or receive transactions, and their preexisting
// accounts become archived.
func ArchiveManagerNode(ctx context.Context, mnodeID string) error {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer dbtx.Rollback(ctx)

	const accountQ = `UPDATE accounts SET archived = true WHERE manager_node_id = $1`
	_, err = pg.FromContext(ctx).Exec(ctx, accountQ, mnodeID)
	if err != nil {
		return errors.Wrap(err, "archiving accounts")
	}

	const mnQ = `UPDATE manager_nodes SET archived = true WHERE id = $1`
	_, err = pg.FromContext(ctx).Exec(ctx, mnQ, mnodeID)
	if err != nil {
		return errors.Wrap(err, "archive query")
	}
	return dbtx.Commit(ctx)
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
	_, err := pg.FromContext(ctx).Exec(ctx, q, managerNodeID, pg.Strings(xpubs))
	return err
}

func managerNodeVariableKeys(ctx context.Context, managerNodeID string) (int, error) {
	const q = `SELECT variable_keys FROM manager_nodes WHERE id = $1`
	count := 0
	err := pg.FromContext(ctx).QueryRow(ctx, q, managerNodeID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
