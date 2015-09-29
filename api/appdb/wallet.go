package appdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
)

// Errors returned by CreateAssetGroup.
// May be wrapped using package chain/errors.
var (
	ErrBadLabel     = errors.New("bad label")
	ErrBadXPubCount = errors.New("bad xpub count")
	ErrBadXPub      = errors.New("bad xpub")
)

// Wallet represents a single wallet. It is intended to be used wth API
// responses.
type Wallet struct {
	ID          string        `json:"id"`
	Blockchain  string        `json:"blockchain"`
	Label       string        `json:"label"`
	Keys        []*hdkey.XKey `json:"keys,omitempty"`
	SigsReqd    int           `json:"signatures_required,omitempty"`
	PrivateKeys []*hdkey.XKey `json:"private_keys,omitempty"`
}

// InsertWallet inserts a new wallet into the database.
func InsertWallet(ctx context.Context, appID, label string, keys, gennedKeys []*hdkey.XKey) (w *Wallet, err error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction
	const q = `
		INSERT INTO manager_nodes (label, project_id, generated_keys)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	xprvs := keysToStrings(gennedKeys)
	err = pg.FromContext(ctx).QueryRow(q, label, appID, pg.Strings(xprvs)).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "insert wallet")
	}

	err = createRotation(ctx, id, keysToStrings(keys)...)
	if err != nil {
		return nil, errors.Wrap(err, "create rotation")
	}

	return &Wallet{
		ID:          id,
		Blockchain:  "sandbox",
		Label:       label,
		Keys:        keys,
		SigsReqd:    1,
		PrivateKeys: gennedKeys,
	}, nil
}

// Balance is a struct describing the balance of
// an asset that a wallet or bucket has.
type Balance struct {
	AssetID   string `json:"asset_id"`
	Confirmed int64  `json:"confirmed"`
	Total     int64  `json:"total"`
}

// GetWallet returns basic information about a single wallet.
func GetWallet(ctx context.Context, walletID string) (*Wallet, error) {
	var (
		q     = `SELECT label, block_chain FROM manager_nodes WHERE id = $1`
		label string
		bc    string
	)
	err := pg.FromContext(ctx).QueryRow(q, walletID).Scan(&label, &bc)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "wallet ID: %v", walletID)
	}
	if err != nil {
		return nil, err
	}

	return &Wallet{ID: walletID, Label: label, Blockchain: bc}, nil
}

// WalletBalance fetches the balances of assets contained in this wallet.
// It returns a slice of Balances and the last asset ID in the page.
// Each Balance contains an asset ID, a confirmed balance,
// and a total balance. The total and confirmed balances
// are currently the same.
func WalletBalance(ctx context.Context, walletID, prev string, limit int) ([]*Balance, string, error) {
	q := `
		SELECT asset_id, sum(amount)::bigint
		FROM utxos
		WHERE manager_node_id=$1 AND ($2='' OR asset_id>$2)
		GROUP BY asset_id
		ORDER BY asset_id
		LIMIT $3
	`
	rows, err := pg.FromContext(ctx).Query(q, walletID, prev, limit)
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

// ListWallets returns a list of wallets contained in the given application.
func ListWallets(ctx context.Context, appID string) ([]*Wallet, error) {
	q := `
		SELECT id, block_chain, label
		FROM manager_nodes
		WHERE project_id = $1
		ORDER BY created_at
	`
	rows, err := pg.FromContext(ctx).Query(q, appID)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var wallets []*Wallet
	for rows.Next() {
		w := new(Wallet)
		err := rows.Scan(&w.ID, &w.Blockchain, &w.Label)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		wallets = append(wallets, w)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return wallets, nil
}

// UpdateManagerNode updates the label of a manager node.
func UpdateManagerNode(ctx context.Context, mnodeID string, label *string) error {
	if label == nil {
		return nil
	}
	const q = `UPDATE wallets SET label = $2 WHERE id = $1`
	db := pg.FromContext(ctx)
	_, err := db.Exec(q, mnodeID, *label)
	return errors.Wrap(err, "update query")
}

func createRotation(ctx context.Context, walletID string, xpubs ...string) error {
	const q = `
		WITH new_rotation AS (
			INSERT INTO rotations (manager_node_id, keyset)
			VALUES ($1, $2)
			RETURNING id
		)
		UPDATE manager_nodes SET current_rotation=(SELECT id FROM new_rotation)
		WHERE id=$1
	`
	_, err := pg.FromContext(ctx).Exec(q, walletID, pg.Strings(xpubs))
	return err
}
