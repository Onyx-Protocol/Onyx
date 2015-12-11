package appdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
)

// ManagerNode represents a single manager node. It is intended to be used wth API
// responses.
type ManagerNode struct {
	ID          string        `json:"id"`
	Blockchain  string        `json:"blockchain"`
	Label       string        `json:"label"`
	Keys        []*hdkey.XKey `json:"keys,omitempty"`
	SigsReqd    int           `json:"signatures_required,omitempty"`
	PrivateKeys []*hdkey.XKey `json:"private_keys,omitempty"`
}

var ErrBadVarKeys = errors.New("Invalid number of variable keys (must be 0 or 1)")

// InsertManagerNode inserts a new manager node into the database.
func InsertManagerNode(ctx context.Context, projID, label string, keys, gennedKeys []*hdkey.XKey, variableKeys, sigsRequired int) (w *ManagerNode, err error) {
	if variableKeys > 1 {
		return nil, ErrBadVarKeys
	}

	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction
	const q = `
		INSERT INTO manager_nodes (label, project_id, generated_keys, variable_keys, sigs_required)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id string
	xprvs := keysToStrings(gennedKeys)
	err = pg.FromContext(ctx).QueryRow(q, label, projID, pg.Strings(xprvs), variableKeys, sigsRequired).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "insert manager node")
	}

	err = createRotation(ctx, id, keysToStrings(keys)...)
	if err != nil {
		return nil, errors.Wrap(err, "create rotation")
	}

	return &ManagerNode{
		ID:          id,
		Blockchain:  "sandbox",
		Label:       label,
		Keys:        keys,
		SigsReqd:    sigsRequired,
		PrivateKeys: gennedKeys,
	}, nil
}

// Balance is a struct describing the balance of
// an asset that a manager node or account has.
type Balance struct {
	AssetID   string `json:"asset_id"`
	Confirmed int64  `json:"confirmed"`
	Total     int64  `json:"total"`
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
			SELECT label, block_chain, keyset, generated_keys
			FROM manager_nodes mn
			JOIN rotations r ON r.id=mn.current_rotation
			WHERE mn.id = $1
		`
		label       string
		bc          string
		pubKeyStrs  []string
		privKeyStrs []string
	)
	err := pg.FromContext(ctx).QueryRow(q, managerNodeID).Scan(
		&label,
		&bc,
		(*pg.Strings)(&pubKeyStrs),
		(*pg.Strings)(&privKeyStrs),
	)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "manager node ID: %v", managerNodeID)
	}
	if err != nil {
		return nil, err
	}

	pubKeys, err := stringsToKeys(pubKeyStrs)
	if err != nil {
		return nil, errors.Wrap(err, "parsing pub keys")
	}

	privKeys, err := stringsToKeys(privKeyStrs)
	if err != nil {
		return nil, errors.Wrap(err, "parsing private keys")
	}

	return &ManagerNode{
		ID:          managerNodeID,
		Label:       label,
		Blockchain:  bc,
		Keys:        pubKeys,
		PrivateKeys: privKeys,
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
				FROM utxos WHERE manager_node_id=$1 AND asset_id=$2
					AND ($3='' OR account_id>$3)
			UNION ALL
			SELECT 0 AS confirmed, amount AS unconfirmed, account_id
				FROM pool_outputs po WHERE manager_node_id=$1 AND asset_id=$2
					AND ($3='' OR account_id>$3)
				AND NOT EXISTS(
					SELECT 1 FROM pool_inputs pi
					WHERE po.tx_hash = pi.tx_hash AND po.index = pi.index
				)
			UNION ALL
			SELECT 0 AS confirmed, amount*-1 AS unconfirmed, account_id
				FROM utxos u WHERE manager_node_id=$1 AND asset_id=$2
					AND ($3='' OR account_id>$3)
				AND EXISTS(
					SELECT 1 FROM pool_inputs pi
					WHERE u.txid = pi.tx_hash AND u.index = pi.index
				)
		) AS bals
		GROUP BY account_id
		ORDER BY account_id ASC
		LIMIT $4
	`
	rows, err := pg.FromContext(ctx).Query(q, mnodeID, assetID, prev, limit)
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

// ListManagerNodes returns a list of manager nodes contained in the given project.
func ListManagerNodes(ctx context.Context, projID string) ([]*ManagerNode, error) {
	q := `
		SELECT id, block_chain, label
		FROM manager_nodes
		WHERE project_id = $1
		ORDER BY id
	`
	rows, err := pg.FromContext(ctx).Query(q, projID)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var managerNodes []*ManagerNode
	for rows.Next() {
		m := new(ManagerNode)
		err := rows.Scan(&m.ID, &m.Blockchain, &m.Label)
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
	_, err := db.Exec(q, mnodeID, *label)
	return errors.Wrap(err, "update query")
}

// DeleteManagerNode deletes the manager node but only if no
// activity/accounts/addresses/rotations are associated with it
// (enforced by ON DELETE NO ACTION).
func DeleteManagerNode(ctx context.Context, mnodeID string) error {
	const q = `DELETE FROM manager_nodes WHERE id = $1`
	db := pg.FromContext(ctx)
	result, err := db.Exec(q, mnodeID)
	if err != nil {
		if pg.IsForeignKeyViolation(err) {
			return errors.WithDetailf(ErrCannotDelete, "manager node ID %v", mnodeID)
		}
		return errors.Wrap(err, "delete query")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "delete query")
	}
	if rowsAffected == 0 {
		return errors.WithDetailf(pg.ErrUserInputNotFound, "manager node ID %v", mnodeID)
	}
	return nil
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
	_, err := pg.FromContext(ctx).Exec(q, managerNodeID, pg.Strings(xpubs))
	return err
}

func managerNodeVariableKeys(ctx context.Context, managerNodeID string) (int, error) {
	const q = `SELECT variable_keys FROM manager_nodes WHERE id = $1`
	count := 0
	err := pg.FromContext(ctx).QueryRow(q, managerNodeID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
