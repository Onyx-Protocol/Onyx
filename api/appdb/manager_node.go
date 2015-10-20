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

// InsertManagerNode inserts a new manager node into the database.
func InsertManagerNode(ctx context.Context, projID, label string, keys, gennedKeys []*hdkey.XKey) (w *ManagerNode, err error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction
	const q = `
		INSERT INTO manager_nodes (label, project_id, generated_keys)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	xprvs := keysToStrings(gennedKeys)
	err = pg.FromContext(ctx).QueryRow(q, label, projID, pg.Strings(xprvs)).Scan(&id)
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
		SigsReqd:    1,
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

// ManagerNodeBalance fetches the balances of assets contained in this manager node.
// It returns a slice of Balances and the last asset ID in the page.
// Each Balance contains an asset ID, a confirmed balance,
// and a total balance. The total and confirmed balances
// are currently the same.
func ManagerNodeBalance(ctx context.Context, managerNodeID, prev string, limit int) ([]*Balance, string, error) {
	q := `
		SELECT asset_id, sum(amount)::bigint
		FROM utxos
		WHERE manager_node_id=$1 AND ($2='' OR asset_id>$2)
		GROUP BY asset_id
		ORDER BY asset_id
		LIMIT $3
	`
	rows, err := pg.FromContext(ctx).Query(q, managerNodeID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "balance query")
	}
	defer rows.Close()
	var (
		bals []*Balance
		last string
	)

	for rows.Next() {
		var (
			assetID string
			bal     int64
		)
		err = rows.Scan(&assetID, &bal)
		if err != nil {
			return nil, "", errors.Wrap(err, "row scan")
		}
		bals = append(bals, &Balance{assetID, bal, bal})
		last = assetID
	}
	if err = rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "rows error")
	}
	return bals, last, err
}

// ListManagerNodes returns a list of manager nodes contained in the given project.
func ListManagerNodes(ctx context.Context, projID string) ([]*ManagerNode, error) {
	q := `
		SELECT id, block_chain, label
		FROM manager_nodes
		WHERE project_id = $1
		ORDER BY created_at
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
